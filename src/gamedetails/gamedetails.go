package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

type GameDetails struct {
	AppId    int
	Name     string
	Icon     string
	Features []string
}

func getDb() (*dynamodb.DynamoDB, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1")},
	)

	if err != nil {
		return nil, err
	}

	// Create DynamoDB client
	svc := dynamodb.New(sess)
	return svc, nil
}

func getGameDetailsFromCache(appId int, db *dynamodb.DynamoDB) (*GameDetails, error) {
	result, err := db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("steamfilter-gamecache"),
		Key: map[string]*dynamodb.AttributeValue{
			"AppId": {
				N: aws.String(strconv.Itoa(appId)),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	item := GameDetails{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &item)

	if err != nil {
		return nil, err
	}

	if item.AppId == 0 {
		return nil, nil
	}

	return &item, nil
}

func putGameDetailsToCache(details GameDetails, db *dynamodb.DynamoDB) error {
	av, err := dynamodbattribute.MarshalMap(details)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("steamfilter-gamecache"),
	}

	_, err = db.PutItem(input)

	return err
}

func formatDetails(details []GameDetails) (string, error) {
	js, err := json.Marshal(details)
	/*
		var buf bytes.Buffer
		body, err := json.Marshal(dict)
		json.HTMLEscape(&buf, body)
	*/
	return string(js), err
}

func parseGameDetails(appId int, reader io.Reader) (GameDetails, error) {
	var details GameDetails

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return details, err
	}

	features := doc.Find("#category_block .game_area_details_specs").Map(func(i int, s *goquery.Selection) string {
		return s.Text()
	})

	details = GameDetails{
		AppId:    appId,
		Name:     doc.Find(".apphub_AppName").Text(),
		Icon:     fmt.Sprintf("https://steamcdn-a.akamaihd.net/steam/apps/%v/capsule_184x69.jpg", appId),
		Features: features,
	}

	return details, nil
}

func createStoreUrl(appId int) string {
	url := fmt.Sprintf("https://store.steampowered.com/app/%v/", appId)
	return url
}

func fetchGameDetails(appId int, db *dynamodb.DynamoDB) (GameDetails, error) {
	var details GameDetails

	if db != nil {
		cachedDetails, err := getGameDetailsFromCache(appId, db)
		if err != nil {
			return details, err
		}

		if cachedDetails != nil {
			return *cachedDetails, nil
		}
	}

	url := createStoreUrl(appId)
	fmt.Printf("Fetching store page %s\n", url)
	res, err := http.Get(url)

	if err != nil {
		return details, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return details, fmt.Errorf("Steam API response code for fetching game list: %v (url: %v)", res.StatusCode, url)
	}

	details, err = parseGameDetails(appId, res.Body)
	if err != nil {
		return details, err
	}

	if db != nil {
		putGameDetailsToCache(details, db)
	}

	return details, err
}

func createResponse(status int, body string, origin string) Response {
	return Response{
		StatusCode:      status,
		IsBase64Encoded: false,
		Body:            body,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": origin,
			"X-Content-Type-Options":      "nosniff",
		},
	}
}

func GetGameDetails(ctx context.Context, request Request) (Response, error) {
	allAppIds := request.QueryStringParameters["appId"]
	skipCache := request.QueryStringParameters["skipCache"]
	origin := request.Headers["origin"]
	if allAppIds == "" {
		return createResponse(418, "No appIds specified", origin), nil
	}

	appIds := strings.Split(allAppIds, ",")

	var db *dynamodb.DynamoDB
	if skipCache == "" {
		var err error
		db, err = getDb()
		if err != nil {
			return createResponse(500, err.Error(), origin), nil
		}
	}

	var allDetails []GameDetails
	for _, appIdString := range appIds {
		appId, _ := strconv.Atoi(appIdString)
		if appId == 0 {
			continue
		}

		details, err := fetchGameDetails(appId, db)
		allDetails = append(allDetails, details)
		if err != nil {
			return createResponse(418, err.Error(), origin), nil
		}
	}

	body, err := formatDetails(allDetails)
	if err != nil {
		return createResponse(418, err.Error(), origin), nil
	}

	return createResponse(200, body, origin), nil
}

func main() {
	lambda.Start(GetGameDetails)
}
