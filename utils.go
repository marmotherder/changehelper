package main

import (
	"errors"

	"github.com/manifoldco/promptui"
)

func captureMultiLineInput(query, queryContinue, label string, obj *[]string) error {
	queryItems := []string{"No", "Yes"}

	querySelect := promptui.Select{
		Label: query,
		Items: queryItems,
	}

	_, queryResult, err := querySelect.Run()
	if err != nil {
		sLogger.Errorf("failed to capture query from label: '%s'", query)
		return err
	}

	if queryResult == "Yes" {
		for {
			prompt := promptui.Prompt{
				Label: label,
				Validate: func(input string) error {
					if input == "" {
						return errors.New("no text entered")
					}
					return nil
				},
			}

			result, err := prompt.Run()
			if err != nil {
				sLogger.Errorf("failed to capture query from label: '%s'", label)
				return err
			}

			*obj = append(*obj, result)

			queryContinueSelect := promptui.Select{
				Label: queryContinue,
				Items: queryItems,
			}

			_, queryContinueResults, err := queryContinueSelect.Run()
			if err != nil {
				sLogger.Errorf("failed to capture query from label: '%s'", queryContinueResults)
				return err
			}

			if queryContinueResults != "Yes" {
				break
			}
		}
	}

	return nil
}

func mustCaptureMultiLineInput(query, queryContinue, label string, obj *[]string) {
	if err := captureMultiLineInput(query, queryContinue, label, obj); err != nil {
		sLogger.Fatal(err.Error())
	}
}
