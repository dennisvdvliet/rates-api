package main

import (
    "encoding/json"
    "encoding/csv"
    "log"
    "net/http"
    "github.com/gorilla/mux"
    "os"
    "bufio"
    "io"
    "strings"
    "strconv"
    "time"
)

// our main function
func main() {
  router := mux.NewRouter()
  router.Use(middleware)
  PopulateRates()

  port := os.Getenv("PORT")

  if port == "" {
    log.Fatal("$PORT must be set")
  }

  router.HandleFunc("/v1/rates/{from}/{to}/{date}", GetRate).Methods("GET")
  log.Fatal(http.ListenAndServe(":" + port, router))
}

func middleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
    w.Header().Add("Content-Type", "application/json")
    next.ServeHTTP(w,r)
  }) 
}

type Rate struct {
  From  string  `json:"from"`
  To    string  `json:"to"`
  Date  int64   `json:"date"`
  Rate  float64  `json:"rate"`
}

type Message struct {
  Message string `json:"message"`
}

var currencyList []string

var db map[string]map[int64]Rate

func GetRate(w http.ResponseWriter, r *http.Request) {
  var rate Rate
  var dateIndex int64
  params := mux.Vars(r)
  to := strings.ToUpper(params["to"])

  layout := "20060102"
  dateTime, _ := time.Parse(layout, params["date"])

  for i := 0; i < 4; i++ {
    dateIndex, _ = strconv.ParseInt(dateTime.Format(layout), 10, 0)
    if db[to][dateIndex].From == "EUR" {
      rate = db[to][dateIndex]
      break
    }
    dateTime = dateTime.Add(time.Hour * -24)
  }

  if rate.From == "EUR" {
    rate.Date = dateIndex
    rate.To = to
    json.NewEncoder(w).Encode(rate)
    return
  }
  w.WriteHeader(404)
  json.NewEncoder(w).Encode(Message{Message: "No rate found"})
}

func PopulateRates() {
  csvFile, _ := os.Open("csv/rates.csv")
  reader := csv.NewReader(bufio.NewReader(csvFile))
  var currencies []string

  db = make(map[string]map[int64]Rate)
  for {
    line, error := reader.Read()
    if error == io.EOF {
        break
    }
    if error != nil {
      log.Fatal(error)
    }
    if currencies == nil {
      currencies = line
      for i, v := range currencies {
        if i != 0 {
          currencyList = append(currencyList, strings.TrimSpace(v))
          db[strings.TrimSpace(v)] = make(map[int64]Rate)
        }
      }
    } else {
      for i, v := range currencies {
        if i != 0 {
          dateString := strings.Replace(line[0], "-", "", -1)
          date, _ := strconv.ParseInt(dateString, 10, 0)
          rate, _ := strconv.ParseFloat(line[i], 32)
          db[v][date] =  Rate{From: "EUR", Rate: rate}
        }
      }
    }
  }
}
