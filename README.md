# AI Trivia Game

An interactive command-line trivia game powered by OpenAI's GPT model. The program generates customized trivia questions based on user-specified topics, or random topics (if the topic prompt is left blank). The topics can range from very broad (e.g. sports) to very precise (e.g. 16th century French politics). It also provides intelligent answer validation.

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/go-ai-trivia.git
   cd go-ai-trivia
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Set up your OpenAI API key either:
   - Create a `config.json` file with `{"apikey": "your-api-key-from-config"}`
   - Or provide it via command line flag:
      ```bash
      go run main.go --apiKey=your-api-key
      ```

## Usage

Run the program:
   ```bash
   go run main.go
   ```

Or with API key flag (if you did not create a .env file containing your API key):
   ```bash
   go run main.go -apiKey=your-api-key
   ```

Follow the prompts to:
1. Enter a topic (or leave blank for random topics)
2. Answer the generated questions
3. See your final score

## Answer Validation

The program uses a combination of:
- Word matching for multi-word answers (e.g. William Shakespeare)
- Jaro-Winkler distance algorithm for typo tolerance
- Case-insensitive comparison