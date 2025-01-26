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

	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/xrash/smetrics"
)

type TriviaResponse struct {
	Questions []string `json:"questions"`
	Answers   []string `json:"answers"`
}

func main() {
	log.SetFlags(0)

	apiKey := setApiKey()

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
	var apiKeyFlag string
	flag.StringVar(&apiKeyFlag, "apiKey", "", "OpenAI API key")
	flag.Parse()

	if apiKeyFlag != "" {
		return apiKeyFlag
	}

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	return apiKey
}

func generatePrompt() string {
	// get the trivia topic from the user
	topicPrompt := "Enter a topic for your trivia questions, or leave blank for random topics: "
	userTopic := getUserInput(topicPrompt)
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

func getUserInput(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
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
		userGuess := getUserInput(fmt.Sprintf("\nQuestion %d: %s\nEnter your guess: ", i+1, question))
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
