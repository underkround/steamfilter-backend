package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse

func ExampleScrape(ctx context.Context) (Response, error) {
	// Request the HTML page.
	res, err := http.Get("http://metalsucks.net")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	dict := make(map[string]string)

	// Find the review items
	doc.Find(".sidebar-reviews article .content-block").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		dict[s.Find("a").Text()] = s.Find("i").Text()
	})

	var buf bytes.Buffer
	body, err := json.Marshal(dict)
	json.HTMLEscape(&buf, body)

	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            buf.String(),
		Headers: map[string]string{
			"Content-Type":           "application/json",
			"X-MyCompany-Func-Reply": "hello-handler",
		},
	}

	return resp, nil
}

func main() {
	lambda.Start(ExampleScrape)
}
