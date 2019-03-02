package main

import (
	"context"
	"fmt"
	"io"
	"net/http"

	//"github.com/PuerkitoBio/goquery"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

type GameDetails struct {
	appId string
}

func getGameDetailsFromCache(appId string) (*GameDetails, error) {
	// TODO
	return nil, nil
}

func putGameDetailsToCache(details GameDetails) {
	// TODO
}

func formatDetails(details GameDetails) string {
	// TODO
	return ""
}

func parseGameDetails(reader io.Reader) (GameDetails, error) {
	var details GameDetails

	/*
		doc, err := goquery.NewDocumentFromReader(reader)
		if err != nil {
			return details, err
		}

		/*
			// Find the review items
			doc.Find(".sidebar-reviews article .content-block").Each(func(i int, s *goquery.Selection) {
				// For each item found, get the band and title
				dict[s.Find("a").Text()] = s.Find("i").Text()
			})

			var buf bytes.Buffer
			body, err := json.Marshal(dict)
			json.HTMLEscape(&buf, body)
	*/

	return details, nil
}

func createStoreUrl(appId string) string {
	url := fmt.Sprintf("https://store.steampowered.com/app/%s/", appId)
	return url
}

func fetchGameDetails(appId string) (string, error) {
	if appId == "" {
		return "", fmt.Errorf("No appId given")
	}

	details, err := getGameDetailsFromCache(appId)

	if err != nil {
		return "", err
	}

	if details != nil {
		return formatDetails(*details), nil
	}

	url := createStoreUrl(appId)
	res, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("Steam API response code: %s", res.StatusCode)
	}

	parsedDetails, err := parseGameDetails(res.Body)
	if err != nil {
		return "", err
	}

	putGameDetailsToCache(parsedDetails)

	return formatDetails(parsedDetails), err
}

func createResponse(status int, body string) Response {
	return Response{
		StatusCode:      status,
		IsBase64Encoded: false,
		Body:            body,
		//		Headers: map[string]string{
		//			"Content-Type":           "application/json",
		//			"X-MyCompany-Func-Reply": "hello-handler",
		//		},
	}
}

func GetGameDetails(ctx context.Context, request Request) (Response, error) {
	appId := request.QueryStringParameters["appId"]
	body, err := fetchGameDetails(appId)
	if err != nil {
		return createResponse(418, err.Error()), nil
	}

	return createResponse(200, body), nil
}

func main() {
	lambda.Start(GetGameDetails)
}
