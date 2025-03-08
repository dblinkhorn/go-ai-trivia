package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/viper"
	"github.com/xrash/smetrics"
)

type TriviaResponse struct {
	Questions []string `json:"questions"`
	Answers   []string `json:"answers"`
}

func main() {
	log.SetFlags(0)
	cliApiKey := flag.String("apiKey", "", "API key (overrides config file)")
	flag.Parse()
	configFile := "config.json"
	apiKey := ""

	// Check if the file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Printf("Config file %s does not exist.  I will use the api key flag\n", configFile)
		if *cliApiKey != "" {
			apiKey = *cliApiKey
		} else {
			fmt.Println("No API key provided. Please set the API key in the config file or use the -apikey flag.")
			panic("No API Key")
		}
	} else {
		apiKey = setApiKey()
	}

	mainPrompt := generatePrompt()

	client := openai.NewClient(option.WithAPIKey(apiKey))

	// create a channel to signal when to stop the loading indicator
	done := make(chan bool)

	// start loading indicator
	go loadingIndicator(done)

	completion := generateCompletion(context.TODO(), client, mainPrompt)

	// stop the loading indicator
	done <- true

	var trivia TriviaResponse
	err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &trivia)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}

	runTriviaGame(trivia.Questions, trivia.Answers)
}

func setApiKey() string {
	viper.SetConfigName("config") // Name of config file without extension
	viper.SetConfigType("json")   // Type of config file
	viper.AddConfigPath(".")      // Path to look for the config file
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Error reading config file:", err)
		panic(err)
	}
	apiKey := viper.GetString("apikey")

	return apiKey
}

func generatePrompt() string {
	// get the trivia topic from the user
	topicPrompt := "Enter a topic for your trivia questions, or leave blank for random topics: "
	userTopic := getUserInput(topicPrompt, true)
	if userTopic == "" {
		userTopic = "random topics"
	}

	mainPrompt := fmt.Sprintf(`Generate a series of 20 trivia questions about %s.
	Please respond only in the following valid JSON format, with no extra formatting or text in your response:
	{
		"questions": ["Question 1", "Question 2", ...],
		"answers": ["Answer 1", "Answer 2", ...]
	}
	Do not include any questions where the answer would include special symbols or characters.`, userTopic)

	return mainPrompt
}

func generateCompletion(ctx context.Context, client *openai.Client, prompt string) *openai.ChatCompletion {
	chatCompletion, err := client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			}),
			Model: openai.F(openai.ChatModelGPT4oMini),
		})
	if err != nil {
		panic(err.Error())
	}

	return chatCompletion
}

// prints a spinning indicator until receiving a signal to stop
func loadingIndicator(done chan bool) {
	// spinner animation characters
	chars := []rune{'|', '/', '-', '\\'}

	for {
		select {
		// clear the loading indicator when done
		case <-done:
			fmt.Print("\r                                 \r")
			return
		default:
			for _, char := range chars {
				fmt.Printf("\rGenerating trivia questions... %c", char)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func getUserInput(prompt string, allowEmpty bool) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		input := scanner.Text()
		if input != "" || allowEmpty {
			return input
		}
		fmt.Print("Input cannot be empty. Try again: ")
	}
}

func isCorrectAnswer(userGuess, correctAnswer string) bool {
	normalizedUserGuess := strings.ToLower(strings.TrimSpace(userGuess))
	normalizedCorrectAnswer := strings.ToLower(strings.TrimSpace(correctAnswer))

	// check if the user's guess is a substring of the correct answer
	// this covers edge cases where the correct answer includes "the" as in "The Nile"
	// or answering "Shakespeare" when the answer is "William Shakespeare"
	if strings.Contains(normalizedCorrectAnswer, normalizedUserGuess) {
		return true
	}

	// use Jaro-Winkler distance to compare the user's guess to the correct answer
	// the comparison is forgiving of simple typos and misspellings, etc
	return smetrics.JaroWinkler(normalizedUserGuess, normalizedCorrectAnswer, 0.7, 4) > 0.85
}

func runTriviaGame(questions, answers []string) {
	score := 0
	isNumeric := regexp.MustCompile(`^\d+$`)

	// ask the user to answer the trivia questions one by one
	fmt.Println("\nTrivia Questions:")
	for i, question := range questions {
		userGuess := getUserInput(fmt.Sprintf("\nQuestion %d: %s\nEnter your guess: ", i+1, question), false)
		correctAnswer := answers[i]

		// check if the correct answer is numeric
		if isNumeric.MatchString(correctAnswer) {
			// if it's a numeric answer there must be an exact match with the user's guess
			if userGuess == correctAnswer {
				fmt.Println("\nCorrect!")
				score++
			} else {
				fmt.Printf("\nIncorrect. The answer was %s\n", correctAnswer)
			}
		} else {
			if isCorrectAnswer(userGuess, correctAnswer) {
				fmt.Println("\nCorrect!")
				score++
			} else {
				fmt.Printf("\nIncorrect. The answer was %s\n", correctAnswer)
			}
		}
	}

	fmt.Println("\nYour final score: ", score, " out of ", len(questions))
}
