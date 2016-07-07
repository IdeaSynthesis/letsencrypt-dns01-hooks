package main

import (
	"strings"
	"time"
	"os"
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
	_ "strconv"
)

func main(){
	// load the API key
	api_key := os.Getenv("LINODE_API_KEY")
	
	client := &http.Client{}

	// verify that it works
	req, _ := http.NewRequest("GET", "https://api.linode.com", nil)
	q := req.URL.Query()
	q.Add("api_key", api_key)
	q.Add("api_action", "test.echo")
	q.Add("now", fmt.Sprintf("%v", time.Now()))
	req.URL.RawQuery = q.Encode()
	
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		fmt.Printf("Invalid API key ("+api_key+"): "+resp.Status)
		return
	}
	
	// fine, that worked: get a list of all the domains
	req, _ = http.NewRequest("GET", "https://api.linode.com", nil)
	q = req.URL.Query()
	q.Add("api_key", api_key)
	q.Add("api_action", "domain.list")
	req.URL.RawQuery = q.Encode()
	
	resp, _ = client.Do(req)
	defer resp.Body.Close()
	data, _ = ioutil.ReadAll(resp.Body)
	
	var input interface{}
	_ = json.Unmarshal(data, &input)
	//output, _ := json.MarshalIndent(domains, "", "  ")
	// fmt.Printf("Contents: %v", string(output))
	handler := os.Args[1]
	domain := os.Args[2]
	// filename := os.Args[3]
	token := os.Args[4]
	//fmt.Printf("Handler: %s\n", handler)
	domains := input.(map[string]interface{})["DATA"].([]interface{})
	var zone map[string]interface{}
	for _, v := range domains {
		zone = v.(map[string]interface{})
		if !strings.HasSuffix(domain, zone["DOMAIN"].(string)) {
			zone = nil
			continue
		}else{
			break
		}
	}

	// output, _ := json.MarshalIndent(zone, "", "  ")
	// fmt.Printf("Contents: %v\n",string(output))

	// get the zone ID
	domainid := zone["DOMAINID"].(float64)
	
	// process the requested action
	if handler == "deploy_challenge" {
		target := "_acme-challenge."+domain

		// see if there's already a resource
		req, _ = http.NewRequest("GET", "https://api.linode.com", nil)
		q = req.URL.Query()
		q.Add("api_key", api_key)
		q.Add("api_action", "domain.resource.list")
		q.Add("DomainID", fmt.Sprintf("%v", domainid))
		req.URL.RawQuery = q.Encode()

		resp, _ = client.Do(req)
		defer resp.Body.Close()
		data, _ = ioutil.ReadAll(resp.Body)
		_ = json.Unmarshal(data, &input)
		// resourcelist, _ := json.MarshalIndent(input, "", "  ")
		// fmt.Printf("Contents: %v\n",string(resourcelist))
		resourcelist := input.(map[string]interface{})["DATA"].([]interface{})
		var resource map[string]interface{}
		for _, rv := range resourcelist {
			resource = rv.(map[string]interface{})
			if resource["TARGET"].(string) == target {
				break
			}else{
				resource = nil
				continue
			}
		}

		// resoutput, _ := json.MarshalIndent(resource, "", "  ")
		// fmt.Printf("Contents: %v\n",string(resoutput))		
		req, _ = http.NewRequest("GET", "https://api.linode.com", nil)
		q = req.URL.Query()
		q.Add("api_key", api_key)
		if resource == nil {
			// create
			q.Add("api_action", "domain.resource.create")
		}else{
			// update
			q.Add("api_action", "domain.resource.update")
		}
		q.Add("DomainID", fmt.Sprintf("%v", domainid))
		q.Add("Type", "TXT")
		q.Add("Name", target)
		q.Add("Target", token)
		q.Add("TTL", "60")
		req.URL.RawQuery = q.Encode()

		resp, _ = client.Do(req)
		defer resp.Body.Close()
		
		if resp.StatusCode != 200 {
			data, _ = ioutil.ReadAll(resp.Body)
			fmt.Printf("Error: %s\n", string(data))
		}
	}
}

