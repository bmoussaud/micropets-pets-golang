package pets

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/imroc/req"

	"github.com/carlmjohnson/requests"
	"github.com/gin-gonic/gin"

	. "moussaud.org/pets/internal"
)

var calls = 0

// Pet Structure
type Pet struct {
	Index    int
	Name     string
	Type     string
	Kind     string
	Age      int
	URL      string
	Hostname string
	From     string
	URI      string
}

// Path Structure
type Path struct {
	Service  string
	Hostname string
}

// Pets Structure
type Pets struct {
	Total     int
	Hostname  string
	Hostnames []Path
	Pets      []Pet `json:"Pets"`
}

func setupResponse(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

}

func queryPets(backend string) (Pets, error) {

	var pets Pets
	req.Debug = true
	fmt.Printf("* Connecting backend [%s]\n", backend)
	req, err := http.NewRequest("GET", backend, nil)
	if err != nil {
		return pets, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Expires", "10ms")

	//Inject the opentracing header
	if LoadConfiguration().Observability.Enable {
		fmt.Printf("* Inject the opentracing header \n")
		//opentracing.GlobalTracer().Inject(spanCtx, opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("##########################@ ERROR Connecting backend [%s]\n", backend)
		return pets, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return pets, fmt.Errorf("ReadAll got error %s", err.Error())
	}

	json.Unmarshal(body, &pets)
	return pets, nil
}

func queryPet(backend string) (Pet, error) {

	var pet Pet
	req.Debug = true
	fmt.Printf("#queryPet@ Connecting backend [%s]\n", backend)
	req, err := http.NewRequest("GET", backend, nil)
	if err != nil {
		return pet, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Expires", "10ms")

	//Inject the opentracing header
	if LoadConfiguration().Observability.Enable {
		fmt.Printf("* Inject the opentracing header \n")
		//opentracing.GlobalTracer().Inject(spanCtx, opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("#queryPet@ ERROR Connecting backend [%s]\n", backend)
		return pet, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return pet, fmt.Errorf("ReadAll got error %s", err.Error())
	}

	fmt.Printf("#queryPet@ body [%s]\n", string(body))
	json.Unmarshal(body, &pet)
	fmt.Printf("#queryPet@ body [%+v]\n", pet)
	return pet, nil
}

func readiness_and_liveness(c *gin.Context) {
	c.String(http.StatusOK, "OK\n")
}

func pets(c *gin.Context) {
	//span := NewServerSpan(r, "pets")
	//defer span.Finish()

	setupResponse(c)
	time.Sleep(time.Duration(10) * time.Millisecond)

	config := LoadConfiguration()

	var all Pets
	host, err := os.Hostname()
	if err != nil {
		host = "Unknown"
	}
	path := Path{"pets", host}
	all.Hostnames = []Path{path}

	for i, backend := range config.Backends {
		var URL string
		if strings.HasPrefix(backend.Host, "http") {
			URL = fmt.Sprintf("%s:%s%s", backend.Host, backend.Port, backend.Context)
		} else {
			URL = fmt.Sprintf("http://%s:%s%s", backend.Host, backend.Port, backend.Context)
		}

		fmt.Printf("* Accessing %d %s %s .....\n", i, backend.Name, URL)

		//lookupService(backend.Host)

		pets, err := queryPets(URL)
		if err != nil {
			fmt.Printf("* ERROR * Accessing backend [%s][%s]:[%s]\n", backend.Name, URL, err)
		} else {
			fmt.Printf("* process result\n")
			all.Total = all.Total + pets.Total
			all.Hostnames = append(all.Hostnames, Path{backend.Name, pets.Hostname})
			fmt.Printf("* Hostnames %+v\n", all.Hostnames)
			for _, pet := range pets.Pets {
				pet.Type = backend.Name
				pet.URI = fmt.Sprintf("/pets%s", pet.URI)
				all.Pets = append(all.Pets, pet)
			}
			time.Sleep(time.Duration(pets.Total) * time.Millisecond)
		}
	}

	sort.SliceStable(all.Pets, func(i, j int) bool {
		return all.Pets[i].Name < all.Pets[j].Name
	})

	calls = calls + 1
	if calls%50 == 0 {
		//fmt.Printf("Zero answer from all the services (0) %d\n ", calls)
		all.Total = 0
	}

	if all.Total == 0 {
		fmt.Printf("Zero answer from all the services (1)\n")
		c.JSON(http.StatusInternalServerError, "no answer from all the pets services")
		return
	} else {
		c.IndentedJSON(http.StatusOK, all)
	}
}

func getConfig(c *gin.Context) {
	//span := NewServerSpan(r, "info")
	//defer span.Finish()

	setupResponse(c)
	time.Sleep(time.Duration(10) * time.Millisecond)

	service := c.Param("kind")

	config := LoadConfiguration()

	fmt.Printf("Display a configuration for service ... => %s \n", service)
	for _, backend := range config.Backends {
		if service == backend.Name {
			var URL string
			if strings.HasPrefix(backend.Host, "http") {
				URL = fmt.Sprintf("%s:%s%s", backend.Host, backend.Port, backend.Context)
			} else {
				URL = fmt.Sprintf("http://%s:%s%s", backend.Host, backend.Port, backend.Context)
			}
			URL = strings.Replace(URL, "data", "config", -1)

			fmt.Printf("* Accessing Info %s\t %s\n", backend.Name, URL)

			var pet any
			err := requests.
				URL(URL).
				ToJSON(&pet).
				Fetch(context.Background())
			fmt.Printf("* result pet from queryPet %+v\n", pet)
			if err != nil {
				fmt.Printf("* ERROR * Accessing backend [%s][%s]:[%s]\n", backend.Name, URL, err)
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			} else {
				fmt.Printf("* process result\n")
				c.IndentedJSON(http.StatusOK, pet)
			}
		}
	}
}

func detail(c *gin.Context) {
	//span := NewServerSpan(r, "detail")
	//defer span.Finish()

	setupResponse(c)
	time.Sleep(time.Duration(10) * time.Millisecond)

	service := c.Param("kind")
	id := c.Param("id")

	config := LoadConfiguration()

	fmt.Printf("Display a specific pet with ID ... => %s %s \n", service, id)
	for _, backend := range config.Backends {
		if service == backend.Name {
			var URL string
			if strings.HasPrefix(backend.Host, "http") {
				URL = fmt.Sprintf("%s:%s%s/%s", backend.Host, backend.Port, backend.Context, id)
			} else {
				URL = fmt.Sprintf("http://%s:%s%s/%s", backend.Host, backend.Port, backend.Context, id)
			}

			fmt.Printf("* Accessing %s\t %s\n", backend.Name, URL)
			pet, err := queryPet(URL)
			fmt.Printf("* result pet from queryPet %+v\n", pet)
			if err != nil {
				fmt.Printf("* ERROR * Accessing backend [%s][%s]:[%s]\n", backend.Name, URL, err)
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			} else {
				fmt.Printf("* process result\n")
				pet.Type = backend.Name
				c.IndentedJSON(http.StatusOK, pet)
			}
		}
	}
}

func Start() {

	config := LoadConfiguration()

	if config.Service.Listen {
		r := gin.Default()

		port := config.Service.Port
		r.GET("/pets/liveness", readiness_and_liveness)
		r.GET("/pets/readiness", readiness_and_liveness)

		r.GET("/liveness", readiness_and_liveness)
		r.GET("/readiness", readiness_and_liveness)

		r.GET("/pets", pets)
		r.GET("/pets/:kind/v1/data/:id", detail)
		r.GET("/pets/:kind/config", getConfig)
		r.GET("/", pets)

		fmt.Printf("******* Starting to the Pets service on port %s\n", port)
		for i, backend := range config.Backends {
			fmt.Printf("* Managing %d\t %s\t %s:%s%s\n", i, backend.Name, backend.Host, backend.Port, backend.Context)
		}
		fmt.Printf("> \n")

		r.Run(config.Service.Port)
	} else {
		fmt.Printf("******* Don't Execute Pets service and exit \n")
	}
}
