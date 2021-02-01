package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
)

type Instroom struct {
	Complexiteit string  `json:"COMPLEXITEIT"`
	Dist         string  `json:"Dist"`
	Lambda       float64 `json:"Lambda"`
}

type Wachttijd struct {
	Dist  string  `json:"Dist"`
	Mu    float64 `json:"Mu"`
	Sigma float64 `json:"Sigma"`
	Loc   float64 `json:"Loc"`
	Max   float64 `json:"Max"`
}

type WerktijdIntake struct {
	Complexiteit string  `json:"COMPLEXITEIT"`
	Dist         string  `json:"Dist"`
	Mu           float64 `json:"Mu"`
	Sigma        float64 `json:"Sigma"`
	Loc          float64 `json:"Loc"`
	Max          float64 `json:"Max"`
}

type WerktijdRest struct {
	Complexiteit string  `json:"COMPLEXITEIT"`
	Dist         string  `json:"Dist"`
	Mu           float64 `json:"Mu"`
	Sigma        float64 `json:"Sigma"`
	Loc          float64 `json:"Loc"`
	Max          float64 `json:"Max"`
}

type Aanraakmoment struct {
	Complexiteit string  `json:"COMPLEXITEIT"`
	Dist         string  `json:"Dist"`
	Mu           float64 `json:"Mu"`
	Sigma        float64 `json:"Sigma"`
	Loc          float64 `json:"Loc"`
	Max          float64 `json:"Max"`
}

type Systeem struct {
	Instroomkansen  []Instroom       `json:"INSTROOM"`
	Aanraakmomenten []Aanraakmoment  `json:"AANTAL_AANRAAKMOMENTEN"`
	Intaketijden    []WerktijdIntake `json:"INTAKE_TAAKTIJD"`
	Resttijden      []WerktijdRest   `json:"REST_TAAKTIJD"`
	Wachttijd       Wachttijd        `json:"WERKDAGEN_WACHTTIJD"`
}

type Aanbod struct {
	workday int
	claims  []Claim
}

type Service struct {
	workday int
}

type Claim struct {
	Id                   string `json:"Id"`
	Complexiteit         string `json:"Complexiteit"`
	Actual_aanraakmoment int    `json:"Actual"`
	Final_aanraakmoment  int    `json:"Aanraakmomenten"`
	Activeminutes        []int  `json:"ActiveMinutes"`
	Wachtdagen           []int  `json:"Wachtdagen"`
}

type Agent struct {
	Workdays []int
	Id       int
	Minutes  []int
}

type Pool struct {
	Agents         []Agent
	Instroom       []int
	Uitstroom      []int
	Taakvertraging []int
	AHT            []int
	Doorlooptijd   []int
}

type Simparams struct {
	Instroom        int `json:"Instroom"`
	Aanraaktijd     int `json:"Aanraaktijd"`
	Aanraakmomenten int `json:"Aanraakmomenten"`
	Wachtdagen      int `json:"Wachtdagen"`
	FTE             int `json:"FTE"`
	Productiviteit  int `json:"Productiviteit"`
	Zaak            int `json:"Zaak"`
	Werkdagen       int `json:"Werkdagen"`
	Random          int `json:"Random"`
	Vakantieduur    int `json:"VakantieDuur"`
	VakantieFTE     int `json:"VakantieFTE"`
}

type Simresults struct {
	Instroom  []int
	Uitstroom []int
	//Werkvoorraad []int
	//Werkdag    []int
	Wachttijd    float64
	Capaciteit   []float64
	AHT          float64
	Doorlooptijd float64
}

func (i *Instroom) createAanbod(factor float64) int {
	poisson := distuv.Poisson{i.Lambda, SRC}
	size := int(poisson.Rand() * factor)
	return size
}

func (s *Systeem) ServiceAanraakmomenten(c Claim, factor float64) int {
	//AANRAAKMOMENTEN
	total_aanraakmomenten := 1
	for _, a := range s.Aanraakmomenten {
		if c.Complexiteit == a.Complexiteit {
			lognorm := distuv.LogNormal{a.Mu, a.Sigma, SRC}
			size := lognorm.Rand()
			total_aanraakmomenten = int((size + a.Loc) * factor)
			if total_aanraakmomenten > int(a.Max) {
				total_aanraakmomenten = int(a.Max * factor)
			}
			if total_aanraakmomenten < 1 {
				total_aanraakmomenten = 1
			}

		}
	}

	return total_aanraakmomenten
}
func (s *Systeem) ServiceActiveMinutes(c Claim, factor float64) []int {
	//WERKTIJDEN INTAKE
	var tijden []int
	intaketijd := 1
	for _, a := range s.Intaketijden {
		if c.Complexiteit == a.Complexiteit {
			lognorm := distuv.LogNormal{a.Mu, a.Sigma, SRC}
			size := lognorm.Rand()
			intaketijd = int((size + a.Loc) * factor)
			if intaketijd > int(a.Max) {
				intaketijd = int(a.Max * factor)
			}
			if intaketijd < 1 {
				intaketijd = 1
			}

		}

	}
	tijden = append(tijden, intaketijd)

	//WERKTIJDEN REST
	var resttijden []int
	for _, a := range s.Resttijden {
		if c.Complexiteit == a.Complexiteit {
			lognorm := distuv.LogNormal{a.Mu, a.Sigma, SRC}
			for i := 0; i < c.Final_aanraakmoment-1; i++ {
				size := int((lognorm.Rand() + a.Loc) * factor)
				if size > int(a.Max) {
					size = int(a.Max * factor)
				}
				if size < 1 {
					size = 1
				}
				resttijden = append(resttijden, size)
			}
		}
	}
	tijden = append(tijden, resttijden...)
	return tijden
}

//WACHTTIJDEN
func (s *Systeem) ServiceWachttijden(c Claim, factor float64) []int {
	var total_wachtdagen []int
	size := 0
	a := s.Wachttijd
	lognorm := distuv.LogNormal{a.Mu, a.Sigma, SRC}
	for i := 0; i < c.Final_aanraakmoment-1; i++ {
		size = int((lognorm.Rand() + a.Loc) * factor)
		if size > int(a.Max) {
			size = int(a.Max * factor)
		}
		if size < 1 {
			size = 1
		}
		total_wachtdagen = append(total_wachtdagen, size)
	}
	return total_wachtdagen
}

func (s *Systeem) SimuleerInstroom(werkdag int, p Simparams) []Claim {
	//per complexiteit het aanbod
	var claims []Claim

	factorInstroom := float64(1 + (float64(p.Instroom) / 100.0))
	factorAanraakmomenten := float64(1 + (float64(p.Aanraakmomenten) / 100.0))
	factorAanraaktijd := float64(1 + (float64(p.Aanraaktijd) / 100.0))
	factorWachttijd := float64(1 + (float64(p.Wachtdagen) / 100.0))
	for _, i := range s.Instroomkansen {
		size := i.createAanbod(factorInstroom)
		complexiteit := i.Complexiteit
		for i := 0; i < size; i++ {
			//println(strconv.Itoa(werkdag) + complexiteit + strconv.Itoa(i))
			var c Claim
			c.Complexiteit = complexiteit
			c.Id = strconv.Itoa(werkdag) + complexiteit + strconv.Itoa(i)
			c.Actual_aanraakmoment = 0
			c.Final_aanraakmoment = s.ServiceAanraakmomenten(c, factorAanraakmomenten)
			c.Activeminutes = s.ServiceActiveMinutes(c, factorAanraaktijd)
			c.Wachtdagen = s.ServiceWachttijden(c, factorWachttijd)
			claims = append(claims, c)
		}
	}
	return claims
}
func sum(array []int) int {
	result := 0
	for _, v := range array {
		result += v
	}
	return result
}

func (pool *Pool) pickAgent(random int, werkdag int, overall int) Agent {
	//kies agent met laagste totale werkvoorraad/ meeste tijd over
	smize := 0
	smidx := 0

	if overall == 1 {
		for aidx, agnts := range pool.Agents {
			if sum(agnts.Minutes[werkdag:]) > smize {
				smidx = aidx
				smize = sum(agnts.Minutes)
			}
		}
	} else {
		for aidx, agnts := range pool.Agents {
			if agnts.Minutes[werkdag] > smize {
				smidx = aidx
				smize = agnts.Minutes[werkdag]
			}

		}

	}
	agent := pool.Agents[smidx]
	r := 0
	//wijs random anders randomtoe
	if random == 1 {
		r = rand.Intn(len(pool.Agents))
		agent = pool.Agents[r]
	}
	return agent
}

func (ap *Pool) setVakantie(p Simparams) {
	//Zet als er een vakantie is opgegeven het aantal agents, voor de tijd van vakantie even op 0
	//die dan later niet meemen

	for d := 0; d < p.Vakantieduur; d++ {
		for f := 0; f < p.VakantieFTE; f++ {
			ap.Agents[f].Minutes[d+OPWARMTIJD+10] = 0

		}

	}
}

//aggresive devide function
func (s *Systeem) devideAgents(p Simparams) Pool {

	poolsize := p.FTE
	var pool Pool
	werkfactor := float64(p.Productiviteit) / 100.0

	werkdagen := p.Werkdagen
	//maak pool met agents
	//TODO: variabiliteit in beschikbaarheid per dag
	for i := 0; i < poolsize; i++ {
		a := Agent{}

		for j := 0; j < werkdagen+10000; j++ {
			a.Workdays = append(a.Workdays, i)
			a.Minutes = append(a.Minutes, int(480*werkfactor))
		}
		pool.Agents = append(pool.Agents, a)
	}

	//set de vakantie
	pool.setVakantie(p)

	//init arrays met in/uitstroom
	for i := 0; i < werkdagen; i++ {
		pool.Instroom = append(pool.Instroom, 0)
		pool.Uitstroom = append(pool.Uitstroom, 0)
	}
	//voor elke werkdag verdeel over agents/ random/ suboptimaal
	for i := 0; i < werkdagen; i++ {
		claims := s.SimuleerInstroom(i, p)

		// neem instroom
		pool.Instroom[i] += len(claims)

		for _, c := range claims {

			// neem aht en doorlooptijd van alle claims
			pool.AHT = append(pool.AHT, sum(c.Activeminutes))
			pool.Doorlooptijd = append(pool.Doorlooptijd, sum(c.Wachtdagen))

			//bepaal of claim compleet bij agent hoort of losse taken
			lossetaak := 0
			if rand.Float64() > float64(p.Zaak)/100 {
				lossetaak = 1
				//println("ok")
			}

			//neem een agent voor hele claim
			agent := pool.pickAgent(p.Random, i, 1)

			//voor elke volgende taak
			for d, m := range c.Activeminutes {
				//als losse taak, dan voor elke taak andere agent
				if lossetaak == 1 {
					agent = pool.pickAgent(p.Random, i, 0)
				}

				for j := i; j < werkdagen; j++ {
					//if INTAKE
					if d == 0 {
						if m <= agent.Minutes[j] {
							agent.Minutes[j] -= m
							pool.Taakvertraging = append(pool.Taakvertraging, j-i)
							if len(c.Activeminutes) == 1 && j <= werkdagen {
								pool.Uitstroom[j] += 1
							}
							break
						}
					} else {
						if m <= agent.Minutes[j+c.Wachtdagen[d-1]] {
							//fmt.Printf("%d\n", agent)
							agent.Minutes[j+c.Wachtdagen[d-1]] -= m
							pool.Taakvertraging = append(pool.Taakvertraging, j-i)
							//fmt.Println(j+c.Wachtdagen[d-1], werkdagen, len(c.Activeminutes), d, m)
							//fmt.Printf("%d\n", c.Activeminutes)
							//fmt.Printf("%d\n", c.Wachtdagen)
							if len(c.Activeminutes) == d-1 && j+c.Wachtdagen[d-1] <= werkdagen {
								fmt.Println(len(c.Activeminutes), d)
								pool.Uitstroom[j+c.Wachtdagen[d-1]] += 1
							}

							break
						}

					}

				}

			}

		}

	}
	return pool

}

var SYSTEEM Systeem
var SRC = rand.New(rand.NewSource(uint64(time.Now().UnixNano())))
var OPWARMTIJD = 50 //50 dagen opwarmtijd vande simulatie, voor starten vanaf een baseline ijzeren voorraad

func parseSysteem() {
	cwd, err := os.Getwd()
	checkError(err)
	println(cwd)
	jsonFile, err := os.Open(cwd + "/data/proces_claims.json")
	checkError(err)
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &SYSTEEM)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func (s *Systeem) Simulate(p Simparams) Simresults {
	p.Werkdagen += OPWARMTIJD
	poolResult := s.devideAgents(p)
	simResult := Simresults{}
	simResult.Instroom = poolResult.Instroom
	simResult.Uitstroom = poolResult.Uitstroom
	simResult.Wachttijd = float64(sum(poolResult.Taakvertraging)) / float64(len(poolResult.Taakvertraging))
	// 1 fte is 34 uur// dit moet beter, want iemand werkt wel hele dag nooit 6.8 uur dus even laten staan op 480
	totaldagcap := float64(p.FTE) * (480 * float64(p.Productiviteit) / 100.0)
	for i := 0; i < p.Werkdagen; i++ {
		restcap := 0
		for _, agnt := range poolResult.Agents {
			restcap += agnt.Minutes[i]
		}
		simResult.Capaciteit = append(simResult.Capaciteit, 1.0-float64(restcap)/float64(totaldagcap))
	}

	//aht en doorlooptijd
	simResult.AHT = float64(sum(poolResult.AHT)) / float64(len(poolResult.AHT))
	simResult.Doorlooptijd = float64(sum(poolResult.Doorlooptijd)) / float64(len(poolResult.Doorlooptijd))
	return simResult
}

func doSimulation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	//p := Simparams{}
	//var err error
	Instroom, err := strconv.Atoi(r.FormValue("Instroom"))
	checkError(err)
	Aanraaktijd, err := strconv.Atoi(r.FormValue("Aanraaktijd"))
	checkError(err)
	Aanraakmomenten, err := strconv.Atoi(r.FormValue("Aanraakmomenten"))
	checkError(err)
	Wachtdagen, err := strconv.Atoi(r.FormValue("Wachtdagen"))
	checkError(err)
	FTE, err := strconv.Atoi(r.FormValue("FTE"))
	checkError(err)
	Productiviteit, err := strconv.Atoi(r.FormValue("Productiviteit"))
	checkError(err)
	Zaak, err := strconv.Atoi(r.FormValue("Zaak"))
	checkError(err)
	Werkdagen, err := strconv.Atoi(r.FormValue("Werkdagen"))
	checkError(err)
	Random, err := strconv.Atoi(r.FormValue("Random"))
	checkError(err)
	VakantieDuur, err := strconv.Atoi(r.FormValue("VakantieDuur"))
	checkError(err)
	VakantieFTE, err := strconv.Atoi(r.FormValue("VakantieFTE"))
	checkError(err)
	p := Simparams{Instroom, Aanraaktijd, Aanraakmomenten, Wachtdagen, FTE, Productiviteit, Zaak, Werkdagen, Random, VakantieDuur, VakantieFTE}
	//dec := json.NewDecoder(r.Body)
	//var p Simparams
	//err := dec.Decode(&p)
	//if err != nil {
	//	panic(err)
	//}
	result := SYSTEEM.Simulate(p)
	json.NewEncoder(w).Encode(result)
}

func main() {

	parseSysteem()
	http.HandleFunc("/simulate", doSimulation)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	log.Fatal(http.ListenAndServe(":3000", nil))

}
