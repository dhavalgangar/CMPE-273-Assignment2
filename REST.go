package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"httprouter"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//constants
const (
	DbName       = "mongo_db"
	DbCollection = "location"
)

//Gmaps for storing google maps json data
type Gmaps struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			LocationType string `json:"location_type"`
			Viewport     struct {
				Northeast struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"northeast"`
				Southwest struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"southwest"`
			} `json:"viewport"`
		} `json:"geometry"`
		PartialMatch bool     `json:"partial_match"`
		PlaceID      string   `json:"place_id"`
		Types        []string `json:"types"`
	} `json:"results"`
	Status string `json:"status"`
}

//Req1 from user to create a location
type Req1 struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
}

//Resp1 to user
type Resp1 struct {
	ID         int    `bson:"_id"`
	Name       string `bson:"name"`
	Address    string `bson:"address"`
	City       string `bson:"city"`
	State      string `bson:"state"`
	Zip        string `bson:"zip"`
	Coordinate struct {
		Lat float64 `bson:"lat"`
		Lng float64 `bson:"lng"`
	} `bson:"coordinate"`
}

//Req2 from user to update location
type Req2 struct {
	Address string `json:"address"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
}

//Counter for sequence
type Counter struct {
	ID       string `bson:"_id"`
	Sequence int    `bson:"seq"`
}

func getNextSequence() int {

	//var doc Seq
	session, err := mgo.Dial("mongodb://dhaval:dhaval123@ds041144.mongolab.com:41144/mongo_db")
	checkError(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(DbName).C("counters")

	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"seq": 1}},
		ReturnNew: true,
	}

	doc := Counter{}
	_, err = c.Find(bson.M{"_id": "userid"}).Apply(change, &doc)
	checkError(err)
	fmt.Println(doc.Sequence)
	return doc.Sequence
}

//CreateNewLocation function to create new location
func CreateNewLocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	//accept request and decode to struct from json
	request1 := Req1{}
	response1 := Resp1{}
	json.NewDecoder(r.Body).Decode(&request1)

	// Get json data from google maps api
	var googleMaps Gmaps
	var urlGoogleMaps string
	var buffer bytes.Buffer

	buffer.WriteString("http://maps.google.com/maps/api/geocode/json?address=")
	buffer.WriteString(strings.Replace(request1.Address, " ", "+", -1))
	buffer.WriteString(",+")
	buffer.WriteString(strings.Replace(request1.City, " ", "+", -1))
	buffer.WriteString(",+")
	buffer.WriteString(request1.State)
	buffer.WriteString(",+")
	buffer.WriteString(request1.Zip)
	buffer.WriteString("&sensor=false")
	urlGoogleMaps = buffer.String()

	response, err := http.Get(urlGoogleMaps)

	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)

	if err != nil {
		fmt.Println("error in reading response body")
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	json.Unmarshal([]byte(contents), &googleMaps)

	response1.Coordinate.Lat = googleMaps.Results[0].Geometry.Location.Lat
	response1.Coordinate.Lng = googleMaps.Results[0].Geometry.Location.Lng
	response1.Name = request1.Name
	response1.Address = request1.Address
	response1.City = request1.City
	response1.State = request1.State
	response1.Zip = request1.Zip
	response1.ID = getNextSequence()

	fmt.Println(response1.ID)
	//establish mongodb connection
	session, err := mgo.Dial("mongodb://dhaval:dhaval123@ds041144.mongolab.com:41144/mongo_db")
	checkError(err)
	defer session.Close()
	//insert values in mongo_db
	session.SetMode(mgo.Monotonic, true)
	c := session.DB(DbName).C(DbCollection)
	err = c.Insert(&response1)
	checkError(err)

	// Marshal provided interface into JSON structure
	uj, _ := json.Marshal(response1)

	// Write content-type, statuscode, payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", uj)
}

//GetLocation to get location
func GetLocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	id := p.ByName("idno")
	oid, _ := strconv.Atoi(id)
	//establish mongodb connection
	session, err := mgo.Dial("mongodb://dhaval:dhaval123@ds041144.mongolab.com:41144/mongo_db")
	checkError(err)
	defer session.Close()
	//insert values in mongo_db
	session.SetMode(mgo.Monotonic, true)
	c := session.DB(DbName).C(DbCollection)

	resultResponse := Resp1{}

	err = c.FindId(oid).One(&resultResponse)

	uj, _ := json.Marshal(resultResponse)

	// Write content-type, statuscode, payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", uj)

}

//DeleteLocation to delete a location
func DeleteLocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	id := p.ByName("idno")
	oid, _ := strconv.Atoi(id)
	session, err := mgo.Dial("mongodb://dhaval:dhaval123@ds041144.mongolab.com:41144/mongo_db")
	checkError(err)
	defer session.Close()
	//insert values in mongo_db
	session.SetMode(mgo.Monotonic, true)

	err = session.DB("mongo_db").C("location").RemoveId(oid)
	if err != nil {
		panic(err)
	}

	w.WriteHeader(200)
}

//UpdateLocation to update address of location
func UpdateLocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	id := p.ByName("idno")
	oid, _ := strconv.Atoi(id)
	request2 := Req2{}
	response1 := Resp1{}
	response2 := Resp1{}

	json.NewDecoder(r.Body).Decode(&request2)

	//get coordinates from google maps api
	// Get json data from google maps api
	var googleMaps Gmaps
	var urlGoogleMaps string
	var buffer bytes.Buffer

	buffer.WriteString("http://maps.google.com/maps/api/geocode/json?address=")
	buffer.WriteString(strings.Replace(request2.Address, " ", "+", -1))
	buffer.WriteString(",+")
	buffer.WriteString(strings.Replace(request2.City, " ", "+", -1))
	buffer.WriteString(",+")
	buffer.WriteString(request2.State)
	buffer.WriteString(",+")
	buffer.WriteString(request2.Zip)
	buffer.WriteString("&sensor=false")
	urlGoogleMaps = buffer.String()

	//fmt.Println(urlGoogleMaps)
	response, err := http.Get(urlGoogleMaps)

	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)

	if err != nil {
		fmt.Println("error in reading response body")
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	json.Unmarshal([]byte(contents), &googleMaps)

	//establish mongodb connection
	session, err := mgo.Dial("mongodb://dhaval:dhaval123@ds041144.mongolab.com:41144/mongo_db")
	checkError(err)
	defer session.Close()

	//update values in mongo_db
	//session.SetMode(mgo.Monotonic, true)
	c := session.DB(DbName).C(DbCollection)

	err = c.FindId(oid).One(&response2)
	response2.Address = request2.Address
	response2.City = request2.City
	response2.State = request2.State
	response2.Zip = request2.Zip
	response2.Coordinate.Lat = googleMaps.Results[0].Geometry.Location.Lat
	response2.Coordinate.Lng = googleMaps.Results[0].Geometry.Location.Lng
	err = c.Update(bson.M{"_id": oid}, response2)

	checkError(err)

	//get all values from mongodb
	err = c.FindId(oid).One(&response1)

	uj, _ := json.Marshal(response1)

	// Write content-type, statuscode, payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", uj)
}

//to checkError
func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	r := httprouter.New()

	r.POST("/locations", CreateNewLocation)
	r.GET("/locations/:idno", GetLocation)
	r.PUT("/locations/:idno", UpdateLocation)
	r.DELETE("/locations/:idno", DeleteLocation)
	http.ListenAndServe("localhost:5000", r)
}
