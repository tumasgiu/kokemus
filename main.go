package main

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/gorilla/handlers"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// templ represents a single template
type templateHandler struct {
	debug    bool
	data     interface{}
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
	t.templ.Execute(w, t.data)
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
		launchRawListener()
	} else {
		launchHttpServer()
	}
}

func launchRawListener() {
	var (
		device       string = "eth0"
		snapshotLen int32  = 1024
		promiscuous  bool   = false
		err          error
		timeout time.Duration = 30 * time.Second
		handle       *pcap.Handle
	)

	// Open device
	handle, err = pcap.OpenLive(device, snapshotLen, promiscuous, timeout)
	if err != nil {log.Fatal(err) }
	defer handle.Close()

	// Use the handle as a packet source to process all packets
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		// Process packet here
		fmt.Println(packet)
	}
}

func launchHttpServer() {
	hn, err := os.Hostname()
	if err != nil {
		log.Info("could not get hostname")
	}

	tplData := struct {
		Hostname string
		Message  string
		Color    string
	}{
		Hostname: hn,
		Message:  viper.GetString("message"),
		Color:    viper.GetString("color"),
	}

	http.Handle("/", &templateHandler{
		filename: "index.html",
		debug:    viper.GetBool("debug"),
		data:     tplData,
	})

	err = http.ListenAndServe(":8080", handlers.LoggingHandler(os.Stdout, http.DefaultServeMux))
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}