package main

import (
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var storageLocation = "."

func main() {
	discoverPeers()
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("main() Recovered from: %v\n", err)
		}
	}()

	if len(os.Getenv("STORAGE_LOCATION")) > 0 {
		storageLocation = strings.TrimSpace(os.Getenv("STORAGE_LOCATION"))
	}
	fmt.Printf("Using storage location: %s\n", storageLocation)

	err := LoadBinary(storageLocation+"/snapshot.dat", &records)
	if err != nil {
		fmt.Printf("Error loading snapshot: %v\n", err)
	}
	err = LoadBinary(storageLocation+"/indices.dat", &indices)
	if err != nil {
		fmt.Printf("Error loading indices: %v\n", err)
	}
	err = Load(storageLocation+"/schema.dat", &schema)
	if err != nil {
		fmt.Printf("Error loading schema: %v\n", err)
	}

	r := createRouter()
	registerRoutes(r)

	fmt.Printf("Starting ${CICD_GIT_REPO_NAME} on 0.0.0.0:8080\n")
	r.Run("0.0.0.0:8080")
}

var clientset *kubernetes.Clientset
var inCluster bool = false
var serviceName string = "master-keywest-search"
var serviceNS string = "search"

func betterPanic(message string, args ...string) {
	temp := fmt.Sprintf(message, args)
	fmt.Printf("%s\n\n", temp)
	os.Exit(1)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func discoverPeers() {
	var kubeconfig *string
	home := homeDir()
	if home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Println("Local configuration not found, trying in-cluster configuration.")
		config, err = rest.InClusterConfig()
		if err != nil {
			betterPanic(err.Error())
		}
		inCluster = true
	}

	if inCluster {
		log.Printf("Configured to run in in-cluster mode.\n")
	} else {
		log.Printf("Configured to run in out-of cluster mode.\n")
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		betterPanic(err.Error())
	}

	fsel := fields.OneTermEqualSelector("metadata.name", serviceName).String()
	log.Printf("attempted to watch, %v", fsel)
	watcher, err := clientset.CoreV1().Endpoints(serviceNS).Watch(context.Background(), v1.ListOptions{
		FieldSelector: fsel,
	})
	if err != nil {
		log.Fatalf("failed watching endpoints, %v", err)
	}

	go func() {
		ch := watcher.ResultChan()
		for event := range ch {
			ep, ok := event.Object.(*coreV1.Endpoints)
			if !ok {
				log.Printf("unexpected type %T", ep)
			}
			//var ps []string
			for _, s := range ep.Subsets {
				for _, a := range s.Addresses {
					fmt.Printf("Peer: %v\n", a)
					// ps = append(ps, fmt.Sprintf(
					// 	"%s://%s.%s:%s%s",
					// 	*scheme,
					// 	a.TargetRef.Name,
					// 	a.TargetRef.Namespace,
					// 	*port,
					// 	*path))
				}
			}
			//log.Printf("setting peers %#v", ps)
			//peers.Set(ps...)
		}
	}()
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c != nil && c.Request != nil && "OPTIONS" == c.Request.Method {
			// Added this to debug Anu's OPTIONS requests
			fmt.Printf("\nOPTIONS headers:\n")
			for k, v := range c.Request.Header {
				fmt.Printf("Header: %s\n", k)
				fmt.Printf("Value: ")
				for i, val := range v {
					if i > 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("[%s]", val)
				}
				fmt.Printf("\n")
			}
			fmt.Printf("\n")
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, authorization, content-type, accept, origin, Cache-Control, X-Requested-With, access-control-allow-origin, access-control-allow-credentials, access-control-allow-headers, access-control-allow-methods")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if "OPTIONS" == c.Request.Method {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func createRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors())
	return r
}

func registerRoutes(r *gin.Engine) {
	go func() {
		for true {
			time.Sleep(60 * time.Second)
			fmt.Printf("Saving search index...\n")

			saveIndex()
		}
	}()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"Status":  "OK",
			"Service": "${CICD_GIT_REPO_NAME}",
			"Version": "[${CICD_GIT_REPO_NAME}]-[${CICD_GIT_BRANCH}]-[${CICD_GIT_COMMIT}]-[${CICD_EXECUTION_SEQUENCE}]",
		})
	})

	r.GET("/api/search/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"Status":  "OK",
			"Service": "${CICD_GIT_REPO_NAME}",
			"Version": "[${CICD_GIT_REPO_NAME}]-[${CICD_GIT_BRANCH}]-[${CICD_GIT_COMMIT}]-[${CICD_EXECUTION_SEQUENCE}]",
		})
	})

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"Status": "OK",
		})
	})

	r.PUT("/api/search/ingest", ingest)

	r.POST("/api/search/batch/ingest", batchIngest)

	r.POST("/api/search/update", update)

	r.GET("/api/search/schema", getSchema)

	r.GET("/api/search/stats", getStats)

	r.GET("/api/search/query", search)

	r.GET("/api/search/clear", clearIndex)

	r.DELETE("/api/search/delete/:id", deleteRecord)

	r.Static("/api/search/swaggerui/", "doc/swagger-ui-dist")
	r.StaticFile("/api/search/swagger.json", "./doc/swagger.json")
}

var saving sync.Mutex

func saveIndex() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("saveIndex() Recovered from: %v\n", err)
		}
	}()

	if _, err := os.Stat(storageLocation + "/tmp"); os.IsNotExist(err) {
		err := os.MkdirAll(storageLocation+"/tmp", 0700)
		if err != nil {
			fmt.Printf("Cannot create folder: %s\n", storageLocation+"/tmp")
		}
	}

	mutex.Lock()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		err := SaveBinary(storageLocation+"/tmp/snapshot.dat", records)
		if err != nil {
			fmt.Printf("Error saving snapshot: %v\n", err)
		}
	}()
	go func() {
		defer wg.Done()
		err := SaveBinary(storageLocation+"/tmp/indices.dat", indices)
		if err != nil {
			fmt.Printf("Error saving indices: %v\n", err)
			os.Remove(storageLocation + "/tmp/indices.dat")
		}

	}()
	go func() {
		defer wg.Done()
		err := Save(storageLocation+"/tmp/schema.dat", schema)
		if err != nil {
			fmt.Printf("Error saving schema: %v\n", err)
			os.Remove(storageLocation + "/tmp/schema.dat")
		}
	}()

	wg.Wait()

	if _, err := os.Stat(storageLocation + "/tmp/snapshot.dat"); !os.IsNotExist(err) {
		if _, err := os.Stat(storageLocation + "/tmp/indices.dat"); !os.IsNotExist(err) {
			if _, err := os.Stat(storageLocation + "/tmp/schema.dat"); !os.IsNotExist(err) {

				err := os.Rename(storageLocation+"/tmp/snapshot.dat", storageLocation+"/snapshot.dat")
				if err != nil {
					fmt.Printf("Cannot move snapshot.dat: %v\n", err)
					os.Remove(storageLocation + "/tmp/snapshot.dat")
				}

				err = os.Rename(storageLocation+"/tmp/indices.dat", storageLocation+"/indices.dat")
				if err != nil {
					fmt.Printf("Cannot move indices.dat: %v\n", err)
				}

				err = os.Rename(storageLocation+"/tmp/schema.dat", storageLocation+"/schema.dat")
				if err != nil {
					fmt.Printf("Cannot move schema.dat: %v\n", err)
				}
			}
		}
	}
	mutex.Unlock()
}
