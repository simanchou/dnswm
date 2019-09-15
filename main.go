package main

import (
	"dnswm/utils"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	BucketName   = "domains"
	TimeFormat   = "2006-01-02 15:04:05"
	ZoneFilePath = "/opt/goproject/src/dnswm/zones"
)

type Domain struct {
	Name      string
	Serial    int64
	Records   map[string]*RecordEntry
	CreatedAt string
}

type RecordEntry struct {
	ID       string
	Name     string
	Type     string
	TTL      int
	Priority int
	Value    string
}

type NameSorter []*RecordEntry

func (a NameSorter) Len() int           { return len(a) }
func (a NameSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a NameSorter) Less(i, j int) bool { return a[i].Name < a[j].Name }

var db = &bolt.DB{}

func init() {
	var err error
	db, err = bolt.Open("my.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("open db fail, error: %s", err)
	}

	// init db
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(BucketName))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatalf("init db fail, error: %s", err)
	}

	// check db
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		if b == nil {
			log.Fatalf("init db fail, can not find bucket which named %s\n", BucketName)
		}
		return nil
	})
	log.Println("begin to start dns web manager")

}

func main() {
	// close db when app exit
	defer db.Close()

	// init some data to db
	/*
		d := Domain{
			Name:"example.lan",
			Serial:1,
			Records: map[string][]RecordEntry{
				"A":{
					{
						Name:"www",
						Type:"A",
						TTL:3600,
						Priority:-1,
						Value:"127.0.0.1",
					},
				},
			},
		}

		err = db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(BucketName))
			encoded, err := json.Marshal(d)
			if err != nil {
				return err
			}
			return b.Put([]byte(d.Name), encoded)
		})
		if err != nil {
			log.Fatalln(err)
		}

	*/
	/*
		td := NewDomain("t1")
		tr, err := NewRecordEntry()
		if err != nil {
			log.Println(err)
		}
		tr.Name = "www"
		tr.Type = "A"
		tr.TTL = 600
		tr.Priority = -1
		tr.Value = "1.1.1.1"
		td.Records = map[string]*RecordEntry{
			tr.ID: tr,
		}

		td.AddRecordEntry("w2", "a", "2.2.2.2", 300, -1)
		td.AddRecordEntry("w3", "a", "3.3.3.3", 300, -1)

		fmt.Printf("%#v\n", td)

		err = td.SaveToDB()
		if err != nil {
			log.Println(err)
		}
		err = td.GenZoneFile()
		if err != nil {
			log.Println(err)
		}

	*/

	d, _ := DomainFromDB("t1.lan")
	err := d.DelRecordEntry("8d676ee2-d144-4cd4-a0bd-b737c3795cc2")
	if err != nil {
		log.Println(err)
	} else {
		d.SaveToDB()
		d.GenZoneFile()
	}

	ds, _ := GetAllDomain()
	for _, i := range ds {
		fmt.Printf("Domain: %s\n", i.Name)
		var _r []*RecordEntry
		for _, v := range i.Records {
			_r = append(_r, v)
		}

		sort.Sort(NameSorter(_r))

		for _, i := range _r {
			fmt.Printf("\tID: %s\tName: %s\tValue: %s\n", i.ID, i.Name, i.Value)
		}
	}

	// static file, such as css,js,images
	staticFiles := http.FileServer(http.Dir("assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", staticFiles))

	//http.HandleFunc("/", domainList)
	//http.HandleFunc("/domaindel", domainDel)
	//http.HandleFunc("/record", recordList)
	//http.HandleFunc("/recorddel", recordDel)

	http.HandleFunc("/api/domain", APIDomain)
	http.HandleFunc("/api/record", APIRecord)

	http.ListenAndServe(":9001", nil)
}

func GetAllDomain() (domains []Domain, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			_d := Domain{}
			err := json.Unmarshal(v, &_d)
			if err != nil {
				return err
			}
			domains = append(domains, _d)
		}
		return nil
	})
	return
}

func NewDomain(name string) *Domain {
	return &Domain{
		Name:      name,
		Serial:    1,
		CreatedAt: time.Now().Format(TimeFormat),
	}
}

func DomainFromDB(name string) (d Domain, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		_d := b.Get([]byte(name))

		if _d == nil {
			err = fmt.Errorf("no such domain")
			return err
		}

		err = json.Unmarshal(_d, &d)
		if err != nil {
			return err
		}

		return nil
	})

	return
}

func (d *Domain) SaveToDB() (err error) {
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		encoded, err := json.Marshal(d)
		if err != nil {
			return err
		}
		return b.Put([]byte(d.Name), encoded)
	})
	return
}

func (d *Domain) DelDomainFromDB() (err error) {
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		err = b.Delete([]byte(d.Name))
		return err
	})
	return
}

func (d *Domain) GenZoneFile() (err error) {
	fileContent := []string{
		fmt.Sprintf("$ORIGIN %s.", d.Name),
		fmt.Sprintf("@\t3600\tIN\tSOA\tns1.mydns.local.\troot( %d 2H 30M 2W 1D )", d.Serial),
		fmt.Sprintf("@\t3600\tIN\tNS\tns1.mydns.local.")}

	for _, record := range d.Records {
		var r string
		if record.Type == "MX" {
			r = fmt.Sprintf("%s\t%d\tIN\t%s\t%d\t%s", record.Name, record.TTL, record.Type, record.Priority, record.Value)
		} else {
			r = fmt.Sprintf("%s\t%d\tIN\t%s\t%s", record.Name, record.TTL, record.Type, record.Value)
		}

		fileContent = append(fileContent, r)
	}

	fileName := fmt.Sprintf("/opt/goproject/src/dnswm/zones/%s", d.Name)
	fmt.Println(fileName)
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0666)
	defer f.Close()

	fc := strings.Join(fileContent, "\n")

	_, err = f.Write([]byte(fc))
	if err != io.EOF {
		return err
	}
	return nil
}

func (d *Domain) DelZoneFile() (err error) {
	filePath := path.Join(ZoneFilePath, d.Name)
	err = os.Remove(filePath)
	return err
}

func (d *Domain) RecordIsExist(str string) (isExist bool) {
	isExist = false
	if _, ok := d.Records[str]; ok {
		isExist = true
	}
	return isExist
}

func (d *Domain) AddRecordEntry(rName, rType, rValue string, rTTL, rPriority int) {
	d.Serial += 1
	r := RecordEntry{}
	rType = strings.ToUpper(rType)
	//rValue = strings.TrimSpace(rValue)
	rValue = strings.Join(strings.Fields(rValue), "")
	if rType == "CNAME" {
		_last := rValue[len(rValue)-1:]
		if _last != "." {
			rValue = rValue + "."
		}
	}

	r.ID = utils.MD5ID(rName + rType + rValue)
	r.Name = rName
	r.Type = rType

	if rTTL < 300 {
		r.TTL = 300
	} else {
		r.TTL = rTTL
	}

	r.Value = rValue

	if rType == "MX" && rPriority > 0 {
		r.Priority = rPriority
	} else {
		r.Priority = -1
	}

	d.Records[r.ID] = &r
}

func (d *Domain) DelRecordEntry(id string) (err error) {
	if _, ok := d.Records[id]; ok {
		delete(d.Records, id)
		d.Serial += 1
		return nil
	}

	return fmt.Errorf("no record entry")
}

func DomainValidate(name string) (ok bool, err error) {
	ok = true
	err = nil
	if !strings.Contains(name, ".") {
		ok = false
		err = fmt.Errorf("domain %s is unspport type, only support *.lan", name)
		return
	} else {
		_dl := strings.Split(name, ".")
		if len(_dl) != 2 || _dl[len(_dl)-1] != "lan" {
			ok = false
			err = fmt.Errorf("domain %s is unspport type, only support *.lan", name)
			return
		}
	}

	return
}

type HTTPResponseData struct {
	Code int
	Msg  string
	Data interface{}
}

type DomainList struct {
	Name      string
	CreatedAt string
}

// api domain
func APIDomain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		_d, err := GetAllDomain()
		if err != nil {
			rd := HTTPResponseData{
				Code: 1,
				Msg:  fmt.Sprintf("%s", err),
				Data: nil,
			}
			json.NewEncoder(w).Encode(rd)
		}

		var dls []DomainList
		for _, i := range _d {
			dl := DomainList{}
			dl.Name = i.Name
			dl.CreatedAt = i.CreatedAt

			dls = append(dls, dl)

		}
		rd := HTTPResponseData{
			Code: 0,
			Data: dls,
		}

		json.NewEncoder(w).Encode(rd)

	case "POST":
		r.ParseForm()
		_d := r.Form["domain"][0]
		log.Println(_d)

		rd := HTTPResponseData{
			Code: 0,
			Msg:  fmt.Sprintf("domain %s add successful", _d),
		}

		if ok, err := DomainValidate(_d); !ok {
			rd.Msg = fmt.Sprintf("%s", err)
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}

		_, err := DomainFromDB(_d)
		if err == nil {
			rd.Code = 1
			rd.Msg = fmt.Sprintf("domain %s is exist", _d)
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}

		domainForAdd := NewDomain(_d)
		domainForAdd.Records = make(map[string]*RecordEntry)
		log.Printf("%#v\n", domainForAdd)
		err = domainForAdd.SaveToDB()
		if err != nil {
			rd.Code = 1
			rd.Msg = fmt.Sprintf("%s", err)
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}
		err = domainForAdd.GenZoneFile()
		if err != nil {
			rd.Code = 1
			rd.Msg = fmt.Sprintf("%s", err)
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}

		log.Println(rd.Msg)
		json.NewEncoder(w).Encode(rd)
	case "DELETE":
		r.ParseForm()
		_d := r.Form["domain"][0]
		log.Println(_d)

		rd := HTTPResponseData{
			Code: 0,
			Msg:  fmt.Sprintf("domain %s delete successful", _d),
		}
		domainForDel, err := DomainFromDB(_d)
		if err != nil {
			rd = HTTPResponseData{
				Code: 1,
				Msg:  fmt.Sprintf("%s", err),
			}
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}

		err = domainForDel.DelDomainFromDB()
		if err != nil {
			rd = HTTPResponseData{
				Code: 1,
				Msg:  fmt.Sprintf("%s", err),
			}
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}

		err = domainForDel.DelZoneFile()
		if err != nil {
			rd = HTTPResponseData{
				Code: 1,
				Msg:  fmt.Sprintf("%s", err),
			}
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}

		log.Println(rd.Msg)
		json.NewEncoder(w).Encode(rd)

	default:
		rd := HTTPResponseData{
			Code: 1,
			Msg:  fmt.Sprintf("unknow method"),
			Data: nil,
		}
		json.NewEncoder(w).Encode(rd)
	}
}

// API Record
func APIRecord(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		p := r.URL.Query()
		_d := p.Get("domain")

		rd := HTTPResponseData{Code: 0}
		if ok, err := DomainValidate(_d); !ok {
			rd.Msg = fmt.Sprintf("%s", err)
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}
		domain, err := DomainFromDB(_d)
		if err != nil {
			rd.Msg = fmt.Sprintf("%s", err)
			json.NewEncoder(w).Encode(rd)
			return
		}
		rd.Data = domain
		log.Printf("query records for domain %s done", _d)
		json.NewEncoder(w).Encode(rd)
	case "POST":
		r.ParseForm()

		pl := []string{"domain", "name", "type", "ttl", "value", "priority"}
		pm := map[string]string{}
		for _, p := range pl {
			pm[p] = r.Form.Get(p)
		}

		rd := HTTPResponseData{
			Code: 0,
			Msg:  fmt.Sprintf("add record for domain %s successful", pm["domain"]),
		}

		for k := range pm {
			if k == "domain" || k == "name" || k == "type" || k == "value" {
				if pm[k] == "" {
					rd.Msg = fmt.Sprintf("miss some args, like %s", k)
					log.Println(rd.Msg)
					json.NewEncoder(w).Encode(rd)
					return
				}
			}

			if k == "type" && strings.ToUpper(pm[k]) == "MX" {
				if pm["priority"] == "" {
					rd.Msg = fmt.Sprintf("miss proirity for type mx")
					log.Println(rd.Msg)
					json.NewEncoder(w).Encode(rd)
					return
				}
			}
		}

		d, err := DomainFromDB(pm["domain"])
		if err != nil {
			rd.Msg = fmt.Sprintf("%s", err)
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}
		_ttl, err := strconv.Atoi(pm["ttl"])
		if err != nil {
			rd.Msg = fmt.Sprintf("%s", err)
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}
		var _priority int
		if strings.ToUpper(pm["type"]) == "MX" {
			_priority, err = strconv.Atoi(pm["priority"])
			if err != nil {
				rd.Msg = fmt.Sprintf("%s", err)
				log.Println(rd.Msg)
				json.NewEncoder(w).Encode(rd)
				return
			}
		}

		log.Printf("%#v\n", d)

		_id := utils.MD5ID(pm["name"] + pm["type"] + pm["value"])
		if d.RecordIsExist(_id) {
			rd.Msg = fmt.Sprintf("[name: %s, type: %s, value: %s] record is exist", pm["name"], pm["type"], pm["value"])
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}

		d.AddRecordEntry(pm["name"], pm["type"], pm["value"], _ttl, _priority)
		err = d.SaveToDB()
		if err != nil {
			rd.Msg = fmt.Sprintf("%s", err)
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}
		err = d.GenZoneFile()
		if err != nil {
			rd.Msg = fmt.Sprintf("%s", err)
			log.Println(rd.Msg)
			json.NewEncoder(w).Encode(rd)
			return
		}

		log.Println(rd.Msg)
		json.NewEncoder(w).Encode(rd)
	default:
		rd := HTTPResponseData{
			Code: 1,
			Msg:  fmt.Sprintf("unknown method"),
		}
		log.Println(rd.Msg)
		json.NewEncoder(w).Encode(rd)
	}
}
