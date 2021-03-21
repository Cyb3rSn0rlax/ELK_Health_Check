package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	//"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7"
	"golang.org/x/crypto/ssh/terminal"
)

func display() {
	const logo = `

  _____  _      _  __
 | ____|| |    | |/ /
 |  _|  | |    | ' /
 | |___ | |___ | . \
 |_____||_____||_|\_\
  _   _               _  _    _
 | | | |  ___   __ _ | || |_ | |__
 | |_| | / _ \ / _' || || __|| '_ \
 |  _  ||  __/| (_| || || |_ | | | |
 |_| |_| \___| \__,_||_| \__||_| |_|
   ____  _                  _
  / ___|| |__    ___   ___ | | __
 | |    | '_ \  / _ \ / __|| |/ /
 | |___ | | | ||  __/| (__ |   <
  \____||_| |_| \___| \___||_|\_\

====================================
	v0.1 by @Cyb3rSn0rlax
====================================

This is a basic script that runs multiple checks for an Elasticsearch cluster health. 
It looks for unavailable nodes and checks indices status (Green, Yellow, Red) also 
verifies cluster's health and allocations and looks for unassigned shards; creates 
and writes everything to txt files.

  `
	fmt.Println(Logo(logo))

}

//username, password, _ := credentials()
func credentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", "", err
	}

	password := string(bytePassword)
	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}

func main() {
	display()
	fmt.Print("Enter ELK Master IP Address: ")
	var address string
	fmt.Scanln(&address)
	fmt.Print("Enter number of cluster nodes: ")
	var nodesNum uint
	fmt.Scanln(&nodesNum)
	username, password, _ := credentials()

	dt := time.Now()
	var nameFolder = "ELKHealthCheck-" + dt.Format("01-02-2006-15-04-05")
	folderName, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	err = os.Mkdir(folderName+"\\"+nameFolder, 0755)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("")
	requestURL("https://"+address+":9200/_cat/nodes?v", "Nodes Status", username, password, "NodesStatus", nameFolder, folderName)
	goGrep("NodesStatus", nameFolder, folderName, "", "NodesStatus", nodesNum)
	log.Println(strings.Repeat("~", 37))
	requestURL("https://"+address+":9200/_cat/indices?v", "Indices Status", username, password, "IndicesStatus", nameFolder, folderName)
	goGrep("IndicesStatus", nameFolder, folderName, "yellow", "IndicesStatus", nodesNum)
	goGrep("IndicesStatus", nameFolder, folderName, "red", "IndicesStatus", nodesNum)
	log.Println(strings.Repeat("~", 37))
	requestURL("https://"+address+":9200/_cluster/health?v", "Cluster Health", username, password, "ClusterHealth", nameFolder, folderName)
	log.Println(strings.Repeat("~", 37))
	requestURL("https://"+address+":9200/_cat/allocation?v", "Allocations", username, password, "Allocations", nameFolder, folderName)
	log.Println(strings.Repeat("~", 37))
	requestURL("https://"+address+":9200/_cat/shards?h=index,shard,prirep,state,unassigned.reason", "Unassigned Shards", username, password, "UnassignedShards", nameFolder, folderName)
	goGrep("UnassignedShards", nameFolder, folderName, "UNASSIGNED", "UnassignedShards", nodesNum)
	log.Println(strings.Repeat("~", 37))
	//fmt.Println("\n#####  Disk Usage #####")
	//fmt.Printf("Usage : ")
	//diskSpace()
	//fmt.Printf("Size : ")
	//diskSpaceGigs()
	fmt.Println()
	elasticAPI(username, password, address)

}

func elasticAPI(username, password, address string) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			"https://" + address + ":9200",
		},
		Username: username,
		Password: password,
		Transport: &http.Transport{
			ResponseHeaderTimeout: time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	var (
		r map[string]interface{}
	)
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}
	res, err := es.Info()
	defer res.Body.Close()
	if res.IsError() {
		log.Fatalf("Error: %s", res.String())
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	// Print client and server version numbers.
	//log.Printf("Client: %s", elasticsearch.Version)
	//log.Println(es.Info())
	//log.Printf("Server: %s", r["version"].(map[string]interface{})["number"])
	//log.Println(strings.Repeat("~", 37))
}

func requestURL(url, title, username, password, fileName, nameFolder, folderName string) {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(username, password)
	log.Println(Outp("--> Looking for " + title + " : "))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		if err != nil {
			log.Fatal(err)
		}
		fullName := folderName + "\\" + nameFolder + "\\" + "\\" + fileName + ".txt"
		f, err := os.Create(fullName)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		io.Copy(f, resp.Body)

	}
}

//df -h | awk '{print $5}' | head -n 2 | tail -1 | sed 's/[\.%-]//g'
//df -h | awk '{print $4}' | head -n 2 | tail -1 | sed 's/[\.%-]//g'

func diskSpace() {
	df := exec.Command("df", "-h")
	awk := exec.Command("awk", "{print $5}")
	head := exec.Command("head", "-n", "2")
	tail := exec.Command("tail", "-1")
	sed := exec.Command("sed", "s/[\\.%-]//g")

	awk.Stdin, _ = df.StdoutPipe()
	head.Stdin, _ = awk.StdoutPipe()
	tail.Stdin, _ = head.StdoutPipe()
	sed.Stdin, _ = tail.StdoutPipe()
	sed.Stdout = os.Stdout
	sed.Stderr = os.Stderr
	_ = sed.Start()
	_ = tail.Start()
	_ = head.Start()
	_ = awk.Start()
	_ = df.Run()
	_ = sed.Wait()
	_ = tail.Wait()
	_ = head.Wait()
	_ = awk.Wait()

}

func diskSpaceGigs() {
	df := exec.Command("df", "-h")
	awk := exec.Command("awk", "{print $4}")
	head := exec.Command("head", "-n", "2")
	tail := exec.Command("tail", "-1")
	sed := exec.Command("sed", "s/[\\.%-]//g")

	awk.Stdin, _ = df.StdoutPipe()
	head.Stdin, _ = awk.StdoutPipe()
	tail.Stdin, _ = head.StdoutPipe()
	sed.Stdin, _ = tail.StdoutPipe()
	sed.Stdout = os.Stdout
	sed.Stderr = os.Stderr
	_ = sed.Start()
	_ = tail.Start()
	_ = head.Start()
	_ = awk.Start()
	_ = df.Run()
	_ = sed.Wait()
	_ = tail.Wait()
	_ = head.Wait()
	_ = awk.Wait()

}

var (
	Info = Teal
	Warn = Yellow
	Fata = Red
	Outp = Green
	Logo = Purple
)

var (
	Black   = Color("\033[1;30m%s\033[0m")
	Red     = Color("\033[1;31m%s\033[0m")
	Green   = Color("\033[1;32m%s\033[0m")
	Yellow  = Color("\033[1;33m%s\033[0m")
	Purple  = Color("\033[1;34m%s\033[0m")
	Magenta = Color("\033[1;35m%s\033[0m")
	Teal    = Color("\033[1;36m%s\033[0m")
	White   = Color("\033[1;37m%s\033[0m")
)

func Color(colorString string) func(...interface{}) string {
	sprint := func(args ...interface{}) string {
		return fmt.Sprintf(colorString,
			fmt.Sprint(args...))
	}
	return sprint
}

func readFile(file string, to chan<- string) {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	freader := bufio.NewReader(f)
	for {
		line, er := freader.ReadBytes('\n')
		if er == nil {
			to <- string(line)
		} else {
			break
		}

	}
	close(to)
}

func grepLine(pat string, from <-chan string, result chan<- bool) {
	var wg sync.WaitGroup

	for line := range from {
		wg.Add(1)

		go func(l string) {
			defer wg.Done()
			if strings.Contains(l, pat) {
				result <- true
			}
		}(string(line))
	}

	wg.Wait()
	close(result)
}

func goGrep(fileName, nameFolder, folderName, pat, task string, nodesNum uint) {
	//file, pat := "IndicesStatus.txt", "UNASSIGNED"
	textChan := make(chan string, 10)
	resultChan := make(chan bool, 10)
	fullName := folderName + "\\" + nameFolder + "\\" + "\\" + fileName + ".txt"
	go readFile(fullName, textChan)
	go grepLine(pat, textChan, resultChan)

	var total uint = 0
	for r := range resultChan {
		if r == true {
			total++
		}
	}

	if task == "UnassignedShards" {
		if total != 0 {
			log.Printf(Warn("--> [WARN] - Found %d Unassigned Shards"), total)
			log.Printf(Info("--> [INFO] - File successfully saved. See %s"), fullName)
		} else {
			log.Printf(Info("--> [INFO] - No Unassigned Shards found"))
			log.Printf(Info("--> [INFO] - File successfully saved. See %s"), fullName)
		}
	} else if task == "IndicesStatus" {
		if total != 0 {
			if pat == "yellow" {
				log.Printf(Warn("--> [WARN] - Found %d Indices in YELLOW Status"), total)
				log.Printf(Info("--> [INFO] - File successfully saved. See %s"), fullName)
			} else if pat == "red" {
				log.Printf(Fata("--> [ERROR] - Found %d Indices in RED Status"), total)
				log.Printf(Info("--> [INFO] - File successfully saved. See %s"), fullName)
			} else {
				log.Printf(Info("--> [INFO] - All Indicies are GREEN"))
				log.Printf(Info("--> [INFO] - File successfully saved. See %s"), fullName)
			}
		}
	} else if task == "NodesStatus" {
		if nodesNum != total-1 {
			log.Printf(Fata("--> [ERROR] - %d unavailbale node(s) in the cluster"), nodesNum-(total-1))
			log.Printf(Info("--> [INFO] - File successfully saved. See %s"), fullName)
		} else {
			log.Printf(Info("--> [INFO] - No unavailable nodes found"))
			log.Printf(Info("--> [INFO] - File successfully saved. See %s"), fullName)
		}
	}

}

/*

func createFile(folderName, fileName, nameFolder string) {

	fullName := folderName + "\\" + nameFolder + "\\" + "\\" + fileName + ".txt"
	f, err := os.Create(fullName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
}
*/
