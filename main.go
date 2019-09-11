package main

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	BucketName = "domains"
	TimeFormat = "2006-01-02 15:04:05"
)

type manageProcesses struct{}

type Domain struct {
	Name      string
	Serial    int64
	Records   map[string][]RecordEntry
	CreatedAt string
}

type RecordEntry struct {
	Name     string
	Type     string
	TTL      int
	Priority int
	Value    string
}

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
	var err error

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

	td := NewDomain("t1")
	td.Records = map[string][]RecordEntry{
		"A": {
			{
				Name:     "www",
				Type:     "A",
				TTL:      600,
				Priority: -1,
				Value:    "1.1.1.1",
			},
			{
				Name:     "w4",
				Type:     "A",
				TTL:      600,
				Priority: -1,
				Value:    "4.4.4.4",
			},
			{
				Name:     "w5",
				Type:     "A",
				TTL:      600,
				Priority: -1,
				Value:    "5.5.5.5",
			},
		},
		"MX": {
			{
				Name:     "smtp",
				Type:     "MX",
				TTL:      600,
				Priority: 20,
				Value:    "2.2.2.2",
			},
			{
				Name:     "smtp",
				Type:     "MX",
				TTL:      600,
				Priority: -1,
				Value:    "3.3.3.3",
			},
		},
	}

	td.AddRecordEntry("w6", "A", "6.6.6.6", 600, -1)
	td.AddRecordEntry("w7", "CNAME", "s1.abc.com", 600, -1)
	td.AddRecordEntry("w8", "CNAME", "s2.def.com ", 600, -1)
	td.AddRecordEntry("_salkf2asfasf.safaksf", "txt", " laksjfakls lkajfafsaf24124 235626 ", 600, -1)

	err = td.SaveToDB()
	if err != nil {
		log.Println(err)
	}
	err = td.GenZoneFile()
	if err != nil {
		log.Println(err)
	}

	// static file, such as css,js,images
	staticFiles := http.FileServer(http.Dir("assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", staticFiles))

	//http.HandleFunc("/", domainList)
	//http.HandleFunc("/domaindel", domainDel)
	//http.HandleFunc("/record", recordList)
	//http.HandleFunc("/recorddel", recordDel)

	http.ListenAndServe(":9001", nil)
}

func (mp *manageProcesses) GetAll() (domains []Domain, err error) {
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
		Name:      fmt.Sprintf("%s.lan", name),
		Serial:    1,
		CreatedAt: time.Now().Format(TimeFormat),
	}
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

func (d *Domain) GenZoneFile() (err error) {
	fileContent := []string{
		fmt.Sprintf("$ORIGIN %s.", d.Name),
		fmt.Sprintf("@\t3600\tIN\tSOA\tns1.mydns.local.\troot( %d 2H 30M 2W 1D )", d.Serial)}

	for k, v := range d.Records {
		var r string
		if strings.ToUpper(k) == "MX" {
			fmt.Println("--------------", k)
			if len(v) > 1 {
				for _, i := range v {
					if i.Priority < 0 {
						r = fmt.Sprintf("%s\t%d\tIN\t%s\t%d\t%s", i.Name, i.TTL, k, 10, i.Value)
					} else {
						r = fmt.Sprintf("%s\t%d\tIN\t%s\t%d\t%s", i.Name, i.TTL, k, i.Priority, i.Value)
					}
					fileContent = append(fileContent, r)
				}
			} else {
				if v[0].Priority < 0 {
					r = fmt.Sprintf("%s\t%d\tIN\t%s\t%d\t%s", v[0].Name, v[0].TTL, k, 10, v[0].Value)
				} else {
					r = fmt.Sprintf("%s\t%d\tIN\t%s\t%d\t%s", v[0].Name, v[0].TTL, k, v[0].Priority, v[0].Value)
				}
				fileContent = append(fileContent, r)
			}
		} else {
			fmt.Println("++++++++++++++", k)
			fmt.Printf("%#v\n", v)
			if len(v) > 1 {
				for _, i := range v {
					r = fmt.Sprintf("%s\t%d\tIN\t%s\t%s", i.Name, i.TTL, k, i.Value)
					fileContent = append(fileContent, r)
				}
			} else {
				r = fmt.Sprintf("%s\t%d\tIN\t%s\t%s", v[0].Name, v[0].TTL, k, v[0].Value)
				fileContent = append(fileContent, r)
			}
		}
	}

	f, err := os.OpenFile(d.Name, os.O_CREATE|os.O_WRONLY, 0666)
	defer f.Close()

	fc := strings.Join(fileContent, "\r")

	_, err = f.Write([]byte(fc))
	if err != io.EOF {
		return err
	}
	return nil
}

func (d *Domain) AddRecordEntry(rName, rType, rValue string, rTTL, rPriority int) {
	d.Serial += 1
	rType = strings.ToUpper(rType)
	//rValue = strings.TrimSpace(rValue)
	rValue = strings.Join(strings.Fields(rValue), "")
	r := RecordEntry{
		Name: rName,
		Type: rType,
	}

	if rTTL < 300 {
		r.TTL = 300
	} else {
		r.TTL = rTTL
	}

	if rType == "CNAME" {
		_last := rValue[len(rValue)-1:]
		if _last != "." {
			rValue = rValue + "."
		}
	}

	r.Value = rValue

	if rType == "MX" && rPriority > 0 {
		r.Priority = rPriority
	} else {
		r.Priority = -1
	}

	fmt.Printf("%#v\n", r)

	d.Records[rType] = append(d.Records[rType], r)
}
