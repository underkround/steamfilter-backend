package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	AppId       int
	Name        string
	Icon        string
	Features    []string
	Genres      []string
	ReleaseDate int64
	Developer   string
	Publisher   string
	Rating      int
	StoreLink   string
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

	features := doc.Find("#category_block .game_area_details_specs").Not(".learning_about").Map(func(i int, s *goquery.Selection) string {
		return s.Text()
	})

	genres := doc.Find(".block_content .details_block b:contains(Genre)").NextUntil("br").Map(func(i int, s *goquery.Selection) string {
		return s.Text()
	})

	releaseDateString := doc.Find(".date").First().Text()
	var releaseDate int64
	if len(releaseDateString) == 4 {
		releaseDateParsed, _ := time.Parse("2006", releaseDateString)
		releaseDate = releaseDateParsed.Unix()
	} else {
		releaseDateParsed, _ := time.Parse("2 Jan, 2006", releaseDateString)
		releaseDate = releaseDateParsed.Unix()
	}

	developer := doc.Find("b:contains(Developer)").Next().Text()
	publisher := doc.Find("b:contains(Publisher)").Next().Text()

	rating := -1
	ratingString, exists := doc.Find(".user_reviews_summary_row").Last().Attr("data-tooltip-html")
	if exists {
		i := strings.Index(ratingString, "%")
		if i >= 0 {
			ratingz := ratingString[0:i]
			rating, _ = strconv.Atoi(ratingz)
		}
	}

	details = GameDetails{
		AppId:       appId,
		Name:        doc.Find(".apphub_AppName").Text(),
		Icon:        fmt.Sprintf("https://steamcdn-a.akamaihd.net/steam/apps/%v/capsule_184x69.jpg", appId),
		Features:    features,
		Genres:      genres,
		ReleaseDate: releaseDate,
		Developer:   developer,
		Publisher:   publisher,
		Rating:      rating,
		StoreLink:   fmt.Sprintf("https://store.steampowered.com/app/%v/", appId),
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
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, err := client.Get(url)

	if err != nil {
		return details, err
	}

	defer res.Body.Close()

	if res.StatusCode == 302 {
		details.AppId = appId
		putGameDetailsToCache(details, db)
		return details, fmt.Errorf("Game is missing from Steam: %v (url: %v)", res.StatusCode, url)
	}

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
		if err != nil {
			fmt.Printf("Error getting data for AppId %v, %v\n", appId, err.Error())
			continue
		}

		if details.Name != "" {
			allDetails = append(allDetails, details)
		}
	}

	body, err := formatDetails(allDetails)
	if err != nil {
		return createResponse(418, err.Error(), origin), nil
	}

	if body == "null" {
		// HACK THE PLANET
		body = "[]"
	}

	return createResponse(200, body, origin), nil
}

func main() {
	lambda.Start(GetGameDetails)
}
