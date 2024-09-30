package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

var (
	addr      string
	pathfiles string
)
var secretKey = []byte("your-256-bit-secret")

func main() {
	flag.StringVar(&addr, "addr", "", "адрес и порт сервера (в формате server:port)")
	flag.StringVar(&pathfiles, "pathfiles", "", "путь для хранения файлов по умолчанию")
	flag.Parse()

	if addr == "" {
		fmt.Print("Введите адрес и порт сервера (в формате server:port): ")
		var input string
		fmt.Scanln(&input)
		addr = input
	}

	if pathfiles == "" {
		fmt.Print("Введите путь для хранения файлов по умолчанию: ")
		var input string
		fmt.Scanln(&input)
		pathfiles = input
	}
	// if not exist "/" to add
	if !strings.HasPrefix(pathfiles, "/") {
		pathfiles = "/" + pathfiles
	}

	for {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			log.Printf("Неверный формат адреса %s: %s\n", addr, err)
			fmt.Print("Введите адрес и порт сервера (в формате server:port): ")
			var input string
			fmt.Scanln(&input)
			addr = input
			continue
		}

		startPort, err := strconv.Atoi(port)
		if err != nil {
			log.Printf("Неверный порт в адресе %s: %s\n", addr, err)
			fmt.Print("Введите адрес и порт сервера (в формате server:port): ")
			var input string
			fmt.Scanln(&input)
			addr = input
			continue
		}

		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			break
		} else {
			fmt.Printf("Адрес %s недоступен: %s\n", addr, err)
			fmt.Print("Хотите, чтобы программа автоматически нашла доступный порт? (y/n): ")
			var choice string
			fmt.Scanln(&choice)
			if choice == "y" {
				for port := startPort; port < startPort+100; port++ {
					addr = fmt.Sprintf("%s:%d", host, port)
					listener, err := net.Listen("tcp", addr)
					if err == nil {
						listener.Close()
						break
					}
					log.Printf("Адрес %s недоступен: %s\n", addr, err)
				}
				break
			} else {
				fmt.Print("Введите адрес и порт сервера (в формате server:port): ")
				var input string
				fmt.Scanln(&input)
				addr = input
			}
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "File API Server")
	})

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			uploadHandler(w, r)
		} else {
			http.Error(w, "Invalid request method", http.StatusBadRequest)
		}
	})

	http.HandleFunc("/get/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			serveAvatar(w, r)
		} else {
			http.Error(w, "Invalid request method", http.StatusBadRequest)
		}
	})

	// addr = "192.168.0.13:8080"

	log.Printf("Запуск сервера на http://%s\nПуть для хранения файлов по умолчанию: %s", addr, pathfiles)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form
	// if err := r.ParseMultipartForm(10000 << 20); err != nil {
	// 	http.Error(w, "Unable to parse form", http.StatusBadRequest)
	// 	return
	// }

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header missing", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := validateToken(tokenString)
	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Retrieve the file from form data
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Create a directory to store uploaded files if it doesn't exist
	if _, err := os.Stat(pathfiles); os.IsNotExist(err) {
		err := os.Mkdir(pathfiles, os.ModePerm)
		if err != nil {
			http.Error(w, "Failed to create directory", http.StatusInternalServerError)
			return
		}
	}

	// Define the file path
	filename := handler.Filename
	filePath := pathfiles + filename

	// Create the file
	out, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Unable to save the file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	// Copy the file content to the created file
	if _, err = io.Copy(out, file); err != nil {
		http.Error(w, "Failed to write the file", http.StatusInternalServerError)
		return
	}

	// Respond to the client
	avatarURL := addr + "/get" + pathfiles + filename
	fmt.Fprintf(w, avatarURL)
}

func serveAvatar(w http.ResponseWriter, r *http.Request) {
	filepath := r.URL.Path // Map /avatars/filename to ./avatars/filename
	filepath = strings.Replace(filepath, "/get/", "/", 1)
	http.ServeFile(w, r, filepath)
}

func validateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})
}
