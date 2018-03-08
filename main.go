package main

import (
	"net/http"
	"html/template"
	"sync"
	"path/filepath"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
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

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
