// main.go
package main

import (
  "encoding/json"
  "errors"
  "fmt"
  "net/http"
  "os"
  "regexp"
  "strconv"
  "strings"
  "unicode"
)

type Request struct {
  Expression string `json:"expression"`
}

type Response struct {
  Result string `json:"result,omitempty"`
  Error  string `json:"error,omitempty"`
}

func main() {
  http.HandleFunc("/api/v1/calculate", calculateHandler)
  port := ":8081"
  fmt.Printf("Starting server at %s\n", port)
  if err := http.ListenAndServe(port, nil); err != nil {
    fmt.Printf("Server failed: %s\n", err)
    os.Exit(1)
  }
}

func calculateHandler(w http.ResponseWriter, r *http.Request) {
  if r.Method != http.MethodPost {
    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    return
  }

  var req Request
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, "Internal server error", http.StatusInternalServerError)
    return
  }

  result, err := evaluateExpression(req.Expression)
  if err != nil {
    if err.Error() == "invalid expression" {
      writeError(w, "Expression is not valid", http.StatusUnprocessableEntity)
    } else if err.Error() == "division by zero" {
      writeError(w, "Division by zero", http.StatusUnprocessableEntity)
    } else {
      writeError(w, "Internal server error", http.StatusInternalServerError)
    }
    return
  }

  response := Response{Result: result}
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusOK)
  json.NewEncoder(w).Encode(response)
}

func evaluateExpression(expression string) (string, error) {
  expression = strings.ReplaceAll(expression, " ", "")
  validExpr := regexp.MustCompile(`^[0-9+\-*/()]+$`)
  if !validExpr.MatchString(expression) {
    return "", fmt.Errorf("invalid expression")
  }

  result, err := Calc(expression)
  if err != nil {
    return "", err
  }

  return strconv.FormatFloat(result, 'f', -1, 64), nil
}

func Calc(expression string) (float64, error) {
  var numbersStack []float64
  var operationsStack []rune
  var i int
  var leftBrackets int
  var rightBrackets int

  for i < len(expression) {
    if unicode.IsDigit(rune(expression[i])) {
      num, _ := strconv.ParseFloat(string(expression[i]), 64)
      numbersStack = append(numbersStack, num)
      i++
      continue
    }
    switch expression[i] {
    case '+', '-', '*', '/':
      for len(operationsStack) > 0 && priorityOperations(operationsStack[len(operationsStack)-1]) >= priorityOperations(rune(expression[i])) {
        var err error
        numbersStack, operationsStack, err = makeOperation(numbersStack, operationsStack)
        if err != nil {
          return 0, err
        }
      }
      operationsStack = append(operationsStack, rune(expression[i]))
      i++
      continue
    }

    if expression[i] == '(' {
      operationsStack = append(operationsStack, rune(expression[i]))
      leftBrackets++
      i++
      continue
    }

    if expression[i] == ')' {
      rightBrackets++
      if leftBrackets >= rightBrackets {
        for operationsStack[len(operationsStack)-1] != '(' {
          var err error
          numbersStack, operationsStack, err = makeOperation(numbersStack, operationsStack)
          if err != nil {
            return 0, err
          }
        }
        operationsStack = operationsStack[:len(operationsStack)-1]
        leftBrackets--
        rightBrackets--
      } else {
        return 0, errors.New("неправильный ввод")
      }
      i++
      continue
    }
  }

  if leftBrackets != 0 || rightBrackets != 0 {
    return 0, errors.New("неправильный ввод")
  }

  if len(numbersStack)-1 != len(operationsStack) {
    return 0, errors.New("неправильный ввод")
  }

  for len(operationsStack) > 0 {
    var err error
    numbersStack, operationsStack, err = makeOperation(numbersStack, operationsStack)
    if err != nil {
      return 0, err
    }
  }

  return numbersStack[0], nil
}


func makeOperation(numbersStack []float64, operationsStack []rune) ([]float64, []rune, error) {
  if len(numbersStack) < 2 || len(operationsStack) == 0 {
    return numbersStack, operationsStack, errors.New("invalid operation")
  }
  a := numbersStack[len(numbersStack)-2]
  b := numbersStack[len(numbersStack)-1]
  operation := operationsStack[len(operationsStack)-1]

  var result float64
  switch operation {
  case '+':
    result = a + b
  case '-':
    result = a - b
  case '*':
    result = a * b
  case '/':
    if b == 0 {
      return numbersStack, operationsStack, errors.New("division by zero")
    }
    result = a / b
  default:
    return numbersStack, operationsStack, errors.New("unknown operation")
  }

  numbersStack = numbersStack[:len(numbersStack)-2]
  operationsStack = operationsStack[:len(operationsStack)-1]
  return append(numbersStack, result), operationsStack, nil
}

func priorityOperations(operation rune) int {
  switch operation {
  case '*', '/':
    return 2
  case '+', '-':
    return 1
  }
  return 0
}

func writeError(w http.ResponseWriter, message string, statusCode int) {
  response := Response{Error: message}
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(statusCode)
  json.NewEncoder(w).Encode(response)
}
