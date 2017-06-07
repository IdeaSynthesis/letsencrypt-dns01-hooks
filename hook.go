package main

import (
	"bytes"
	"strings"
	"time"
	"os"
	"fmt"
	"net"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"github.com/miekg/dns"
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
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	_ = d.Decode(&input)

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

	// get the zone ID and root
	domainid := zone["DOMAINID"].(json.Number)
	root := zone["DOMAIN"].(string)

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
		d = json.NewDecoder(bytes.NewReader(data))
		d.UseNumber()
		_ = d.Decode(&input)
		// resourcelist, _ := json.MarshalIndent(input, "", "  ")
		// fmt.Printf("Contents: %v\n",string(resourcelist))
		resourcelist := input.(map[string]interface{})["DATA"].([]interface{})
		var resource map[string]interface{}
		for _, rv := range resourcelist {
			resource = rv.(map[string]interface{})
			 // fmt.Printf("Looking For: [%s]; Name: [%s]; Target: [%s]; Type: %s\n", target, resource["NAME"].(string)+"."+root, resource["TARGET"].(string), resource["TYPE"].(string))
			if resource["TYPE"].(string) == "TXT" && resource["NAME"].(string)+"."+root == target {
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
			q.Add("Type", "TXT")
			q.Add("Name", target)
		}else{
			// update
			q.Add("api_action", "domain.resource.update")
			q.Add("ResourceID", fmt.Sprintf("%v", resource["RESOURCEID"].(json.Number)))
		}
		q.Add("DomainID", fmt.Sprintf("%v", domainid))
		q.Add("Target", token)
		q.Add("TTL_sec", "300")
		req.URL.RawQuery = q.Encode()

		resp, _ = client.Do(req)
		defer resp.Body.Close()
		
		if resp.StatusCode != 200 {
			data, _ = ioutil.ReadAll(resp.Body)
			fmt.Printf("Error: %s\n", string(data))
		}else{
			// need to loop: check every 10 seconds to see if the entry has propagated
			var resolver dns.Client
			resolver_host := os.Getenv("DEHYDRATED_RESOLVER")
			if resolver_host != "" {
				resolver = dns.Client{}
			}
			for {
				time.Sleep(10 * time.Second)
				if resolver_host != "" {
					msg := new(dns.Msg)
					msg.Id = dns.Id()
					msg.RecursionDesired = false
					msg.Question = make([]dns.Question, 1)
					msg.Question[0] = dns.Question{target+".", dns.TypeTXT, dns.ClassINET}
					in, _, err1 := resolver.Exchange(msg, resolver_host+":53")
					if err1 == nil {
						if len(in.Answer) > 0 {
							txt := in.Answer[0].(*dns.TXT)
							if txt.Txt[0] == token {
								break
							}
						}
					}
				}else{
					challenges, err1 := net.LookupTXT(target)
					if err1 == nil && len(challenges) > 0 {
						if challenges[0] == token {
							break
						}
					}
				}
			}
		}
	}else if handler == "clean_challenge" {
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
		d = json.NewDecoder(bytes.NewReader(data))
		d.UseNumber()
		_ = d.Decode(&input)
		// resourcelist, _ := json.MarshalIndent(input, "", "  ")
		// fmt.Printf("Contents: %v\n",string(resourcelist))
		resourcelist := input.(map[string]interface{})["DATA"].([]interface{})
		var resource map[string]interface{}
		for _, rv := range resourcelist {
			resource = rv.(map[string]interface{})
			 // fmt.Printf("Looking For: [%s]; Name: [%s]; Target: [%s]; Type: %s\n", target, resource["NAME"].(string)+"."+root, resource["TARGET"].(string), resource["TYPE"].(string))
			if resource["TYPE"].(string) == "TXT" && resource["NAME"].(string)+"."+root == target {
				break
			}else{
				resource = nil
				continue
			}
		}

		// resoutput, _ := json.MarshalIndent(resource, "", "  ")
		// fmt.Printf("Contents: %v\n",string(resoutput))		
		if resource != nil {
			// flush it
			req, _ = http.NewRequest("GET", "https://api.linode.com", nil)
			q = req.URL.Query()
			q.Add("api_key", api_key)
			q.Add("api_action", "domain.resource.delete")
			q.Add("ResourceID", fmt.Sprintf("%v", resource["RESOURCEID"].(json.Number)))
		}
		q.Add("DomainID", fmt.Sprintf("%v", domainid))
		q.Add("Target", token)
		q.Add("TTL_sec", "60")
		req.URL.RawQuery = q.Encode()

		resp, _ = client.Do(req)
		defer resp.Body.Close()
		
		if resp.StatusCode != 200 {
			data, _ = ioutil.ReadAll(resp.Body)
			fmt.Printf("Error: %s\n", string(data))
		}
	}else if handler == "deploy_cert" {
		fmt.Printf("Deploying cert [%s] (full chain %s) and key [%s] for domain [%s]\n", os.Args[4], os.Args[5], os.Args[3], domain)
	}
}
