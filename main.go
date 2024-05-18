package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

var token string
var walletsFilePath string
var initialListPath string
var walletMap map[string]bool = map[string]bool{}

func main() {
	loadENV()
	loadWallets()
	checkAgainstInitialList()
	checkAgainstGithubIssues()
}

func checkAgainstInitialList() {
	fmt.Println("checking if your wallets are flagged in the initial list")
	f, err := os.Open(initialListPath)
	if err != nil {
		log.Fatal("error opening initial list: ", err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("error reading data from initial list: ", err)
	}

	count := 0
	flaggedWallets := []string{}
	for _, rec := range records {
		adrs := cleanUpAddress(rec[0])
		if walletMap[adrs] {
			fmt.Printf("found wallet: %s", rec[0])
			count++
			flaggedWallets = append(flaggedWallets, rec[0])
		}
	}

	if count > 0 {
		fmt.Println("printing flagged wallets list")
		for _, adrs := range flaggedWallets {
			fmt.Println(adrs)
		}
	}
	fmt.Printf("check finished. total wallets flagged: %d\n", count)
}

func checkAgainstGithubIssues() {
	fmt.Println("checking if your wallet has been reported in any github issue")
	// get github client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// summary file
	sfile, err := os.Create("summary.csv")
	if err != nil {
		log.Fatalf("error creating file: %s", err)
	}
	defer sfile.Close()
	sWriter := csv.NewWriter(sfile)

	// make api request
	currentPage := 0
	lastPage := 1
	reportedMap := map[string]bool{}
	for currentPage <= lastPage {
		issues, resp, err := client.Issues.ListByRepo(context.TODO(), "LayerZero-Labs", "sybil-report",
			&github.IssueListByRepoOptions{
				State: "all",
				ListOptions: github.ListOptions{
					Page:    currentPage,
					PerPage: 100,
				},
			})
		if err != nil {
			log.Fatal(err)
		}
		if resp.Response.StatusCode != 200 {
			log.Println("status: ", resp.Response.Status)
		}

		issuesArr := []Issues{}
		for _, issue := range issues {
			if issue == nil || issue.Body == nil {
				continue
			}
			str := *issue.Body
			arr := []string{}
			if len(strings.TrimSpace(str)) > 42 {
				arr = strings.Split(str, "\r\n")
			}

			adrs := []string{}
			for _, a := range arr {
				a = strings.TrimSpace(a)
				if len(a) >= 42 && a[:2] == "0x" {
					adrs = append(adrs, a)
				}
			}
			issu := Issues{
				IssueNumber:       fmt.Sprintf("%d", *issue.Number),
				State:             *issue.State,
				ReportedAddresses: adrs,
				Label:             issue.Labels,
			}
			issuesArr = append(issuesArr, issu)
		}

		// write data to summary file
		for _, issue := range issuesArr {
			// check if our wallet is reported
			count := 0
			reportedWallets := []string{}
			for _, addr := range issue.ReportedAddresses {
				addr = strings.ToLower(addr)
				if walletMap[addr] {
					count++
					reportedWallets = append(reportedWallets, addr)
					reportedMap[addr] = true
				}
			}

			label := concatenateLabels(issue.Label)

			if count > 0 {
				walletStr := "wallet"
				if count > 1 {
					walletStr = "wallets"
				}
				fmt.Printf("%d %s reported in issue: %s, state: %s, label: %s -- %v\n", count, walletStr, issue.IssueNumber, issue.State, label, reportedWallets)
			}

			arr := []string{issue.IssueNumber, issue.State, label, fmt.Sprintf("%d", len(issue.ReportedAddresses)), fmt.Sprintf("%d", count)}
			sWriter.Write(arr)
		}

		lastPage = resp.LastPage
		currentPage++
	}
	sWriter.Flush()
	fmt.Printf("total unique wallets reported: %d\n", len(reportedMap))
	fmt.Println("analysis complete!")
}

func loadWallets() {
	file, err := os.Open(walletsFilePath)
	if err != nil {
		log.Fatal("error opening file: ", err)
	}
	defer file.Close()
	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("error reading csv file")
	}

	for _, rec := range records {
		// validations
		if len(rec) != 1 {
			log.Printf("multiple entries found in a row: %s, ignoring it", rec)
		}
		addr := cleanUpAddress(rec[0])
		if len(addr) != 42 {
			log.Printf("address length is more than 42 characters: %s, ignoring it", rec[0])
		}

		// store in map
		walletMap[addr] = true
	}
	if len(walletMap) == 0 {
		log.Fatal("no valid wallet address found, exiting.")
	}
	fmt.Printf("%d valid addresses found\n", len(walletMap))
}

func loadENV() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	token = os.Getenv("GITHUB_TOKEN")
	walletsFilePath = os.Getenv("WALLETS_FILE_PATH")
	initialListPath = os.Getenv("INITIAL_LIST_PATH")
}

func cleanUpAddress(adrs string) string {
	adrs = strings.TrimSpace(adrs)
	adrs = strings.ToLower(adrs)
	return adrs
}

func concatenateLabels(labels []github.Label) string {
	str := ""
	for _, label := range labels {
		if str == "" {
			str = *label.Name
			continue
		}
		str = str + "-" + *label.Name
	}
	return str
}

type Issues struct {
	IssueNumber       string
	State             string
	ReportedAddresses []string
	Label             []github.Label
}
