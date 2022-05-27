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
	"sync/atomic"
	"time"

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
	Error        error
}

func (m Modulo) Available() bool {
	return len(m.Availability) > 0 && string(m.Availability) != "[]"
}

const (
  xsrfToken = "238013bf-667b-4c19-b87b-41cd60dd988f"
  cookieSignature = "55328d4d2089d20635a8a69fe3b09b46=f8ef8c15e315faf661a36d4c7bc5f983"
)

var (
	cookie      string
	servicio    string
	progress    bool
	sessionFile string

	telegramToken  string
	telegramChatID string
	telegramNotify bool

	qrOnly bool
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
	body, err = io.ReadAll(response.Body)
	if err != nil {
		return false, err
	}

	// Log
	log.Printf("Message '%s' was sent", text)
	log.Printf("Response JSON: %s", string(body))

	// Return
	return true, nil
}

func newLoginReq() (*http.Request, error) {
	req, err := http.NewRequest("GET", "https://citas.sat.gob.mx/api/customLogin", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("authority", "citas.sat.gob.mx")
	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("cookie", fmt.Sprintf("XSRF-TOKEN=%s; %s", xsrfToken, cookieSignature))
	req.Header.Set("referer", "https://citas.sat.gob.mx/creaCita")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-gpc", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.67 Safari/537.36")

	return req, nil
}

func newCalendarReq(entidadeID, moduloID int, sessionToken string) (*http.Request, error) {
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
	req.Header.Set("origin", "https://citas.sat.gob.mx")
	req.Header.Set("referer", "https://citas.sat.gob.mx/creaCita")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-gpc", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.88 Safari/537.36")
	req.Header.Set("x-xsrf-token", fmt.Sprintf("%s", xsrfToken))
	req.Header.Set("cookie", fmt.Sprintf("XSRF-TOKEN=%s; %s; JSESSIONID=%s", xsrfToken, cookieSignature, sessionToken))

	return req, nil
}

func extractSessionToken(cookie string) (string, error) {
	segments := strings.Split(cookie, "; ")
	for _, segment := range segments {
		parts := strings.Split(segment, "=")
		if parts[0] == "JSESSIONID" && len(parts) > 1 {
			return parts[1], nil
		}
	}

	return "", errors.New("session token not found")
}

func login() (string, error) {
	req, err := newLoginReq()
	if err != nil {
		return "", err
	}

	client := http.DefaultClient
	res, err := client.Do(req)

	cookie := res.Header.Get("set-cookie")
	if len(cookie) == 0 {
		return "", errors.New("failed to find login cookie")
	}

	return extractSessionToken(cookie)
}

func readSessionToken(path string) (string, error) {
	f, err := os.Open(path)

	if err == nil {
		var token []byte
		token, err = io.ReadAll(f)
		if err == nil {
			return string(token), nil
		}
	}

	log.Printf("[WARN] failed to read session token: %v, requesting a new one", err)

	token, err := login()
	if err != nil {
		return "", err
	}

	f, err = os.Create(path)
	if err != nil {
		return "", err
	}

	if _, err := f.WriteString(token); err != nil {
		log.Printf("[WARN] failed to store session token: %v", err)
	}

  if err := f.Close(); err != nil {
		log.Printf("[WARN] failed to close file: %v", err)
  }

  log.Printf("[INFO] new session token is stored at %s", sessionFile)

	return token, nil
}

func main() {
	flag.Parse()

	Entidades := entidades()

	sessionToken, err := readSessionToken(sessionFile)

  log.Printf("[INFO] session token is: %s", sessionToken)

	if err != nil {
		log.Printf("[ERR ] failed to obtain session token: %v", err)
		os.Exit(1)
	}

	totalModulos := 0
	for _, entidad := range Entidades {
		totalModulos += len(entidad.Modulos)
	}

	var bar *pb.ProgressBar
	if progress {
		bar = pb.StartNew(totalModulos)
	}

	client := http.Client {
    Timeout: 1 * time.Minute,
  }
	var wg sync.WaitGroup
	wg.Add(totalModulos)

  var sesionExpiredCount int64

	for _, entidad := range Entidades {
		for _, modulo := range entidad.Modulos {
			go func(modulo *Modulo) {
				defer wg.Done()
				if progress {
					defer bar.Increment()
				}

				modulo.Availability = nil

				req, _ := newCalendarReq(entidad.ID, modulo.ID, sessionToken)
				res, err := client.Do(req)

        if err != nil {
          if os.IsTimeout(err) {
            log.Printf("[ERR ] request timeout for: %s, modulo %s", entidad.Name, modulo.Name)
          } else {
            log.Fatalf("failed to send calendar request: %v", err)
				  }
        } else {
          if res.StatusCode == http.StatusOK {
            defer res.Body.Close()
            modulo.Availability, modulo.Error = io.ReadAll(res.Body)
          } else {
            if res.StatusCode != http.StatusNotFound {
              if res.StatusCode == http.StatusInternalServerError {
                log.Printf("[ERR ] request failed for entidad: %s, modulo %s, %s", entidad.Name, modulo.Name, res.Status)
                atomic.AddInt64(&sesionExpiredCount, 1)
              } else {
                fmt.Println("request failed with status code", res.StatusCode)
              }
            }
          }
        }
			}(modulo)
		}
	}

	wg.Wait()

	if progress {
		bar.Finish()
	}

	// New Buffer.
	var b bytes.Buffer

	for _, entidad := range Entidades {
		for _, modulo := range entidad.Modulos {
			if modulo.Error != nil {
				fmt.Printf("%s - %s: %v\n", entidad.Name, modulo.Name, modulo.Error)
			} else {
				message := fmt.Sprintf("%s - %s: %s\n", entidad.Name, modulo.Name, string(modulo.Availability))
				fmt.Print(message)

				if modulo.Available() {
					b.WriteString(message)
				}
			}
		}
		fmt.Println("--------------------------------------------")
	}

	message := b.String()

	// if len(message) == 0 {
	// 	message = "No hay citas disponibles"
	// }

	if telegramNotify {
		if len(message) > 0 {
			if _, err = SendMessage(message); err != nil {
        log.Printf("[ERR ] failed to send Telegram message: %v", err)
			}
		}
	}

  if sesionExpiredCount > 0 {
    log.Printf("[WARN] removing session file %s as server returned 500", sessionFile)
    if err := os.Remove(sessionFile); err != nil {
      log.Printf("[ERR ] failed to remove session file: %v", err)
    }

    message := "SAT server request failed with 500, session cookie will be regenerated"
    if telegramNotify {
      if _, err = SendMessage(message); err != nil {
        log.Printf("[ERR ] failed to send Telegram message: %v", err)
      }
    }
    fmt.Println(message)
    os.Exit(1)
  }
}

func init() {
	flag.StringVar(&sessionFile, "session-file", "./session-token", "file with session token")
	flag.StringVar(&servicio, "servicio", "rfc", "servicio (efirma | rfc)")
	flag.BoolVar(&progress, "progress", false, "show progress")
	flag.StringVar(&telegramToken, "telegram-token", os.Getenv("RFCITA_TOKEN"), "telegram rfcita bot token")
	flag.StringVar(&telegramChatID, "telegram-chat-id", os.Getenv("RFCITA_CHAT_ID"), "telegram user chat id")
	flag.BoolVar(&telegramNotify, "notify", false, "send notification about available citas using Telegram")
	flag.BoolVar(&qrOnly, "qr-only", false, "only check Q.Roo")
}

func entidades() []EntidadFederativa {

	quintanaRoo := EntidadFederativa{
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
	}

	yucatan := EntidadFederativa{
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
	}

	if qrOnly {
		return []EntidadFederativa{
			quintanaRoo,
			yucatan,
		}
	} else {
		return []EntidadFederativa{
			quintanaRoo,
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
}
