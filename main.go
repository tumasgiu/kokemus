package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/gorilla/handlers"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// templ represents a single template
type templateHandler struct {
	debug    bool
	dataMake func() interface{}
	once     sync.Once
	filename string
	templ    *template.Template
}

// ServeHTTP handles the HTTP request.
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if t.debug {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates",
			t.filename)))
	} else {
		t.once.Do(func() {
			t.templ = template.Must(template.ParseFiles(filepath.Join("templates",
				t.filename)))
		})
	}
	t.templ.Execute(w, t.dataMake())
}

func config() {
	viper.SetDefault("debug", false)
	viper.SetDefault("message", "Hello World!")
	viper.SetDefault("color", "red")

	viper.SetEnvPrefix("kokemus")
	viper.AutomaticEnv()

	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	err := viper.ReadInConfig()   // Find and read the config file
	if err != nil { // Handle errors reading the config file
		log.Info("No config file, will use default or env")
	}
}

func main() {
	config()

	log.Info("kokemus")

	if viper.GetString("mode") == "raw" {
		launchRawSniffer()
	}
	launchHttpServer()
}

func launchRawSniffer() {
	var (
		device      string = "eth0"
		snapshotLen int32  = 1024
		promiscuous bool   = false
		err         error
		timeout     time.Duration = 30 * time.Second
		handle      *pcap.Handle
	)

	// Open device
	handle, err = pcap.OpenLive(device, snapshotLen, promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// Use the handle as a packet source to process all packets
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		// Process packet here
		fmt.Println(packet)
	}
}

type DbRecord struct {
	gorm.Model
	Entry string `json:"entry"`
}

var db *gorm.DB

func launchHttpServer() {
	hn, err := os.Hostname()
	if err != nil {
		log.Info("could not get hostname")
	}

	if viper.GetBool("use_db") {
		host := viper.GetString("db_host")
		port := viper.GetString("db_port")
		user := viper.GetString("db_user")
		name := viper.GetString("db_name")
		password := viper.GetString("db_password")
		sslMode := viper.GetString("db_ssl_mode")
		gormArgs := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
			host, port, user, name, password, sslMode)
		log.Info(gormArgs)
		db, err = gorm.Open("postgres", gormArgs)
		if err != nil {
			log.Error(err)
			os.Exit(2)
		}
		db.AutoMigrate(&DbRecord{})
	}

	http.Handle("/", &templateHandler{
		filename: "index.html",
		debug:    viper.GetBool("debug"),
		dataMake: func() interface{} {
			var records []DbRecord
			db.Find(&records)
			return struct {
				Hostname string
				Message  string
				Color    string
				UseDb    bool
				Records  []DbRecord
			}{
				Hostname: hn,
				Message:  viper.GetString("message"),
				Color:    viper.GetString("color"),
				UseDb:    viper.GetBool("use_db"),
				Records:  records,
			}
		},
	})

	http.HandleFunc("/record", recordHandler)

	err = http.ListenAndServe(":8080", handlers.LoggingHandler(os.Stdout, http.DefaultServeMux))
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
	if db != nil {
		db.Close()
	}
}

func recordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not authorized", http.StatusMethodNotAllowed)
		return
	}
	if r.Body == nil || r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Must have a json body", http.StatusBadRequest)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	record := new(DbRecord)
	err = json.Unmarshal(data, &record)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := db.Save(&record).Error; err != nil {
		_, e := w.Write([]byte(fmt.Sprintf(`{ "error": %s }`, strconv.Quote(err.Error()))))
		if e != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	_, e := w.Write([]byte(fmt.Sprintf(`{ "entry": %s }`, strconv.Quote(record.Entry))))
	if e != nil {
		log.Error(e)
	}
}
