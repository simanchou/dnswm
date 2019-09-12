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
	"sort"
	"strings"
	"time"
)

const (
	BucketName = "domains"
	TimeFormat = "2006-01-02 15:04:05"
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

	d.AddRecordEntry("w1", "A", "1.1.1.1", 600, -1)
	d.SaveToDB()
	d.GenZoneFile()

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

	http.ListenAndServe(":9001", nil)
}

func NewRecordEntry() (*RecordEntry, error) {
	uid, err := utils.GenUUID()
	return &RecordEntry{
		ID: uid,
	}, err
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
		Name:      fmt.Sprintf("%s.lan", name),
		Serial:    1,
		CreatedAt: time.Now().Format(TimeFormat),
	}
}

func DomainFromDB(name string) (d Domain, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		_d := b.Get([]byte(name))

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

func (d *Domain) AddRecordEntry(rName, rType, rValue string, rTTL, rPriority int) {
	d.Serial += 1
	rType = strings.ToUpper(rType)
	//rValue = strings.TrimSpace(rValue)
	rValue = strings.Join(strings.Fields(rValue), "")
	if rType == "CNAME" {
		_last := rValue[len(rValue)-1:]
		if _last != "." {
			rValue = rValue + "."
		}
	}

	r, _ := NewRecordEntry()
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

	d.Records[r.ID] = r
}

func (d *Domain) DelRecordEntry(id string) (err error) {
	if _, ok := d.Records[id]; ok {
		delete(d.Records, id)
		d.Serial += 1
		return nil
	}

	return fmt.Errorf("no record entry")
}
