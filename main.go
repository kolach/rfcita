package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/cheggaaa/pb/v3"
)

const ServicioNewRFC = 262
const ServicioFirmaElectronia = 29
const ServicioPersonasMorales = 178

type EntidadFederativa struct {
	ID      int
	Name    string
	Modulos []*Modulo
}

type Modulo struct {
	ID           int
	Name         string
	Availability []byte
}

// MST Cosumel = 262, 146
// MST Chetumal = 262,148
// MST Playa del Carmen = 262, 147
// ADSC Quintana Roo "2" Cancun = 262, 145

var (
	cookie string
  servicio string
  progress bool

  telegramToken string
  telegramChatID string

	Entidades []EntidadFederativa
)

func getUrl() string {
	return fmt.Sprintf("https://api.telegram.org/bot%s", telegramToken)
}

func SendMessage(text string) (bool, error) {
	// Global variables
	var err error
	var response *http.Response

	// Send the message
	url := fmt.Sprintf("%s/sendMessage", getUrl())
	body, _ := json.Marshal(map[string]string{
		"chat_id": telegramChatID,
		"text":    text,
	})
	response, err = http.Post(
		url,
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return false, err
	}

	// Close the request at the end
	defer response.Body.Close()

	// Body
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return false, err
	}

	// Log
	log.Infof("Message '%s' was sent", text)
	log.Infof("Response JSON: %s", string(body))

	// Return
	return true, nil
}

func xsrfToken(cookie string) (string, error) {
	parts := strings.Split(cookie, "; ")
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if kv[0] == "XSRF-TOKEN" {
			return kv[1], nil
		}
	}
	return "", errors.New("failed to find XSRF-TOKEN")
}

func newReq(entidadeID, moduloID int, cookie, xsrf string) (*http.Request, error) {
  servicioID := ServicioNewRFC

  switch servicio {
  case "efirma":
    servicioID = ServicioFirmaElectronia
  case "moral":
    servicioID = ServicioPersonasMorales
  }

	reqBody, _ := json.Marshal(map[string]int{
		"servicio": servicioID,
		"modulo":   moduloID,
	})

	req, err := http.NewRequest("POST", "https://citas.sat.gob.mx/api/slots/calendario", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("authority", "citas.sat.gob.mx")
	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("cookie", cookie)
	req.Header.Set("origin", "https://citas.sat.gob.mx")
	req.Header.Set("referer", "https://citas.sat.gob.mx/creaCita")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-gpc", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.88 Safari/537.36")
	req.Header.Set("x-xsrf-token", xsrf)

	return req, nil
}

func main() {
	flag.Parse()

	xsrf, err := xsrfToken(cookie)
	if err != nil {
		log.Fatalf("%v", err)
	}

  totalModulos := 0
	for _, entidad := range Entidades {
		totalModulos +=  len(entidad.Modulos)
  }

  var bar *pb.ProgressBar
  if progress {
    bar = pb.StartNew(totalModulos)
  }

	client := http.DefaultClient
  var wg sync.WaitGroup
  wg.Add(totalModulos)

	for _, entidad := range Entidades {
		for _, modulo := range entidad.Modulos {
      go func(modulo *Modulo) {
        defer wg.Done()
        if progress {
          defer bar.Increment()
        }

        req, _ := newReq(entidad.ID, modulo.ID, cookie, xsrf)
        res, err := client.Do(req)

        if err != nil {
          log.Fatalf("failed to send calendar request: %v", err)
        }

        defer res.Body.Close()
        body, _ := io.ReadAll(res.Body)

        modulo.Availability = body
      }(modulo)
		}
	}

  wg.Wait()

  if progress {
    bar.Finish()
  }


	for _, entidad := range Entidades {
		for _, modulo := range entidad.Modulos {
			fmt.Printf("%s - %s: %s\n", entidad.Name, modulo.Name, string(modulo.Availability))
		}
		fmt.Println("--------------------------------------------")
	}

}

func init() {
	flag.StringVar(&cookie, "cookie", "", "cookie")
	flag.StringVar(&servicio, "servicio", "rfc", "servicio (efirma | rfc)")
  flag.BoolVar(&progress, "progress", false, "show progress")
	flag.StringVar(&telegramToken, "telegram-token", os.Getenv("RFCITA_TOKEN"), "telegram rfcita bot token")
	flag.StringVar(&telegramChatID, "telegram-chat-id", os.Getenv("RFCITA_CHAT_ID"), "telegram user chat id")

	Entidades = []EntidadFederativa{
		{
			ID:   262,
			Name: "Quintana Roo",
			Modulos: []*Modulo{
				{
					ID:   145,
					Name: "ADSC Quintana Roo '2' Cancun",
				},
				{
					ID:   146,
					Name: "MST Cosumel",
				},
				{
					ID:   147,
					Name: "MST Playa del Carmen",
				},
				{
					ID:   148,
					Name: "MST Chetumal",
				},
			},
		},
    {
			ID:   262,
			Name: "Yucatan",
			Modulos: []*Modulo{
				{
					ID:   193,
					Name: "ADSC Yucatan '1'",
				},
				{
					ID:   194,
					Name: "SARE Merida",
				},
				{
					ID:   195,
					Name: "MST Valladolid",
				},
			},
		}, {
			ID:   262,
			Name: "Guanajuato",
			Modulos: []*Modulo{
				{
					ID:   84,
					Name: "ADSC Guanajuato '3' Celaya",
				},
				{
					ID:   89,
					Name: "ADSC Guanajuato '2' Leon",
				},
				{
					ID:   90,
					Name: "MST Guanajuato",
				},
				{
					ID:   87,
					Name: "MST Irapuato",
				},
				{
					ID:   88,
					Name: "MST Salamanca",
				},
				{
					ID:   86,
					Name: "MST San Migguel de Allende",
				},
			},
		}, {
			ID:   262,
			Name: "Queretaro",
			Modulos: []*Modulo{
				{
					ID:   143,
					Name: "ADSC Quertetaro '1'",
				},
				{
					ID:   144,
					Name: "MST San Juan del Rio",
				},
				{
					ID:   241,
					Name: "MST Boulevares, Queretaro",
				},
			},
		}, {
			ID:   262,
			Name: "Puebla",
			Modulos: []*Modulo{
				{
					ID:   137,
					Name: "MST Huauachinango",
				},
				{
					ID:   140,
					Name: "MST Izucar de Matamoros",
				},
				{
					ID:   142,
					Name: "SARE Atlixco",
				},
				{
					ID:   139,
					Name: "ADSC Puebla '2' Puebla",
				},
				{
					ID:   198,
					Name: "ADSC Puebla '1'",
				},
				{
					ID:   134,
					Name: "MST Palacio Federal",
				},
				{
					ID:   138,
					Name: "Chignahuapan",
				},
				{
					ID:   141,
					Name: "MST San Martin Texmelican",
				},
				{
					ID:   136,
					Name: "MST Teziutlan",
				},
				{
					ID:   135,
					Name: "MST Tehuacan",
				},
			},
		}, {
			ID:   262,
			Name: "Hidalgo",
			Modulos: []*Modulo{
				{
					ID:   98,
					Name: "MST Tula de Allende",
				},
				{
					ID:   96,
					Name: "ADSC Hidalgo '1'",
				},
				{
					ID:   99,
					Name: "MST Tulancingo de Bravo",
				},
				{
					ID:   97,
					Name: "MST Ixmiquilpan",
				},
			},
		}, {
			ID:   262,
			Name: "Morelos",
			Modulos: []*Modulo{
				{
					ID:   119,
					Name: "ADSC Morelos '1'",
				},
				{
					ID:   120,
					Name: "MST Cuaulta Presidencia",
				},
			},
		}, {
			ID:   262,
			Name: "San Luis Potosi",
			Modulos: []*Modulo{
				{
					ID:   150,
					Name: "MST Matehuala",
				},
				{
					ID:   149,
					Name: "ADSC San Luis Potosi '1'",
				},
				{
					ID:   151,
					Name: "MST Cd. Valles",
				},
			},
		}, {
			ID:   262,
			Name: "Ciudad de Mexico",
			Modulos: []*Modulo{
				{
					ID:   70,
					Name: "ADSC Distrito Federal '3' Oriente",
				},
				{
					ID:   68,
					Name: "ADSC Distrito Federal '1' Norte",
				},
				{
					ID:   334,
					Name: "MST Oasis",
				},
				{
					ID:   72,
					Name: "MST Del Valle",
				},
				{
					ID:   66,
					Name: "ASDC Distrito Federal '2' Centro",
				},
				{
					ID:   71,
					Name: "ASDC Distrito Federal '4' Sur",
				},
			},
		}, {
			ID:   262,
			Name: "Chihuahua",
			Modulos: []*Modulo{
				{
					ID:   51,
					Name: "MST Nuevo Cases Grandes",
				},
			},
		}, {
			ID:   262,
			Name: "Baja California",
			Modulos: []*Modulo{
				{
					ID:   37,
					Name: "MST Puerto Penasco",
				},
			},
		},
	}
}
