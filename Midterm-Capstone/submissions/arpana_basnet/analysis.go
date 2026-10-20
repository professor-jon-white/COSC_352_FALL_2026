package main

import (
	"fmt"
	"math"
	"os"
	"sort"
)

func reportTopNotices(
	matched []string,
	requests map[string]int,
	notices map[string]int,
) {
	ranked := append([]string(nil), matched...)

	sort.Slice(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]

		if notices[left] == notices[right] {
			return left < right
		}
		return notices[left] > notices[right]
	})

	limit := 10
	if len(ranked) < limit {
		limit = len(ranked)
	}

	fmt.Println("\nQUESTION 1: Neighborhoods with the most notices")
	fmt.Printf(
		"%-45s %10s %12s\n",
		"Neighborhood", "Notices", "311 Requests",
	)

	for _, neighborhood := range ranked[:limit] {
		fmt.Printf(
			"%-45s %10d %12d\n",
			neighborhood,
			notices[neighborhood],
			requests[neighborhood],
		)
	}

	reportCorrelation(matched, requests, notices)

	if err := reportDumping(matched, notices); err != nil {
		fmt.Fprintln(os.Stderr, "Question 3 error:", err)
		os.Exit(1)
	}

	if err := reportBacklog(notices); err != nil {
		fmt.Fprintln(os.Stderr, "Question 4 error:", err)
		os.Exit(1)
	}

	if err := reportClosureTimes(notices); err != nil {
		fmt.Fprintln(os.Stderr, "Question 5 error:", err)
		os.Exit(1)
	}
}

func reportCorrelation(
	matched []string,
	requests map[string]int,
	notices map[string]int,
) {
	fmt.Println("\nQUESTION 2: Do notice and request counts vary together?")
	fmt.Println("Matched neighborhoods analyzed:", len(matched))

	if len(matched) < 2 {
		fmt.Println("Not enough neighborhoods to calculate correlation.")
		return
	}

	var noticeSum, requestSum float64

	for _, neighborhood := range matched {
		noticeSum += float64(notices[neighborhood])
		requestSum += float64(requests[neighborhood])
	}

	n := float64(len(matched))
	noticeMean := noticeSum / n
	requestMean := requestSum / n

	var crossProduct, noticeSquares, requestSquares float64

	for _, neighborhood := range matched {
		noticeDifference := float64(notices[neighborhood]) - noticeMean
		requestDifference := float64(requests[neighborhood]) - requestMean

		crossProduct += noticeDifference * requestDifference
		noticeSquares += noticeDifference * noticeDifference
		requestSquares += requestDifference * requestDifference
	}

	denominator := math.Sqrt(noticeSquares * requestSquares)
	if denominator == 0 {
		fmt.Println("Correlation is undefined because a count has no variation.")
		return
	}

	correlation := crossProduct / denominator

	fmt.Printf("Pearson correlation: %.3f\n", correlation)
	fmt.Println("Near +1: higher notice counts tend to accompany higher request counts.")
	fmt.Println("Near 0: little linear relationship between the counts.")
	fmt.Println("Near -1: higher notice counts tend to accompany lower request counts.")
	fmt.Println("This measures association, not cause.")
	fmt.Println("Raw counts do not account for neighborhood population or size.")
	fmt.Println("The datasets may represent different time periods.")
}
