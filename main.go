package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/xuri/excelize/v2"
	"gopkg.in/gomail.v2"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

const emailTemplate = `
Flight Status Update:

{{ range $index, $flight := .Flights }}
Flight {{ $flight.FlightNumber }} Status: {{ $flight.Status.Status }}
{{ end }}
`

type FlightStatus struct {
	Status string
}

type Flight struct {
	FlightNumber string
	Status       FlightStatus
}

type TemplateData struct {
	Flights []Flight
}

type flightStatusResponse struct {
	ID   string `json:"-"`
	Data struct {
		Status struct {
			StatusCode        string `json:"statusCode"`
			Status            string `json:"status"`
			Color             string `json:"color"`
			StatusDescription string `json:"statusDescription"`
			Delay             struct {
				Departure struct {
					Minutes int `json:"minutes"`
				} `json:"departure"`
				Arrival struct {
					Minutes int `json:"minutes"`
				} `json:"arrival"`
			} `json:"delay"`
			DelayStatus struct {
				Wording string `json:"wording"`
				Minutes int    `json:"minutes"`
			} `json:"delayStatus"`
			LastUpdatedText string `json:"lastUpdatedText"`
			Diverted        bool   `json:"diverted"`
		} `json:"status"`
	} `json:"data"`
}

func main() {
	xlsx, err := excelize.OpenFile("flight_ids.xlsx")
	if err != nil {
		fmt.Println("Error opening Excel file:", err)
		return
	}

	tomorrow := time.Now().Add(24 * time.Hour)
	tomorrowDateStr := tomorrow.Format("01-02-06")

	flightData := make(map[string]string)
	rows, err := xlsx.GetRows("Sheet1")
	if err != nil {
		fmt.Println("Error reading rows from Excel:", err)
		return
	}
	for _, row := range rows {
		if len(row) >= 3 {
			if tomorrowDateStr == row[2] {
				flightNumber := row[0]
				airlineCode := row[1]

				flightData[flightNumber] = airlineCode
			}
		}
	}
	currentDate := time.Now()
	tomorrowDate := currentDate.Add(24 * time.Hour)
	formattedDate := tomorrowDate.Format("2006/01/02")

	var flights []Flight
	for flightNumber, airlineCode := range flightData {
		flightNumberWithoutCode := strings.Replace(flightNumber, airlineCode, "", -1)

		url := "https://www.flightstats.com/v2/api-next/flight-tracker/" + airlineCode + "/" + flightNumberWithoutCode + "/" + formattedDate
		fmt.Println(url)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error sending request:", err)
			continue
		}

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response:", err)
			continue
		}

		// Parse JSON response into the struct
		var response flightStatusResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			continue
		}

		fmt.Println(response.Data.Status.Status)

		flights = append(flights, Flight{
			FlightNumber: flightNumber,
			Status: FlightStatus{
				Status: response.Data.Status.Status,
			},
		})
	}
	fmt.Println(flights)
	data := TemplateData{
		Flights: flights,
	}

	tmpl := template.Must(template.New("email").Parse(emailTemplate))

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		fmt.Println("Error templating email body", err)
	}

	fmt.Println(buf.String())
	sendEmail(buf.String(), time.Now().Add(24*time.Hour).Format("02-01-06"))
}

func sendEmail(body string, tomorrow string) {
	smtpHost := "smtp.gmail.com"
	smtpPort := 587
	senderEmail := "mehulamba75@gmail.com"
	senderPassword := os.Getenv("EMAIL_PASS")

	// Recipient email address
	recipientEmail := "mehulgohil75@gmail.com"

	// Compose the email
	subject := "Flight Status"

	// Create a new message
	m := gomail.NewMessage()
	m.SetHeader("From", senderEmail)
	m.SetHeader("To", recipientEmail)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	// Create a new dialer
	d := gomail.NewDialer(smtpHost, smtpPort, senderEmail, senderPassword)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	// Send the email
	if err := d.DialAndSend(m); err != nil {
		fmt.Println("Error sending email: ", err)
	} else {
		fmt.Println("Email sent successfully!")
	}
}
