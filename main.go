package main

import (
	"bufio"
	"database/sql"
	// Treiber nur zur Registrierung importieren
	_ "github.com/glebarez/go-sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	htmlTemplate "github.com/gofiber/template/html/v2"
	"gopkg.in/yaml.v3"
	"html"
	"io"
	"math/rand"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var slugRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var (
	titleRe = regexp.MustCompile(`^[^\x00-\x1F<>]{0,100}$`)
	descRe  = regexp.MustCompile(`^[^\x00-\x1F<>]{0,1000}$`)
)

func main() {
	//data-Verzeichnis anlegen (inkl. aller Zwischenordner)
	if err := os.MkdirAll("data/logs", 0777); err != nil {
		log.Fatalf("Fehler beim Erstellen des Datenverzeichnisses: %v", err)
	}
	// Zeitstempel im gewünschten Format generieren
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	// Fehlerprotokolldatei öffnen oder erstellen und Dateinamen zusammensetzen
	filePath := "data/logs/" + timestamp + ".log"
	errorlogfile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		log.Fatalf("Fehler beim Öffnen der Fehlerprotokolldatei: %v", err)
	}
	defer errorlogfile.Close()
	iw := io.MultiWriter(os.Stdout, errorlogfile)
	log.SetOutput(iw)
	//log.SetOutput(errorlogfile)
	log.SetLevel(log.LevelInfo) // Setze das Log-Level
	log.Info("Starting URL Shortener Service Version 2.0...")
	// Konfigurationsdatei laden
	config, err := loadConfig("data/config.yaml")
	if err != nil {
		log.Fatalf("Fehler beim Laden der Konfigurationsdatei: %v", err)
	}
	//domain := config["domain"].(string)
	//config ausgeben
	log.Infof("Konfiguration geladen: %v", config)

	//datenbank initialisieren
	dbpath := "data/data.sqlite"
	db, err := initDB(dbpath)
	if err != nil {
		log.Fatalf("Fehler beim Initialisieren der Datenbank: %v", err)
	} else {
		log.Infof("Datenbank initialisiert: %v", dbpath)
	}
	defer db.Close()

	// blacklist Datei öffnen oder erstellen
	f, err := os.OpenFile("data/domain-blacklist.txt", os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Fatalf("Fehler beim Öffnen und/oder erstellen der Blacklist-Datei: %v", err)
	}
	defer f.Close()
	// Blacklist einlesen

	//blacklist einlesen
	scanner := bufio.NewScanner(f)
	blacklist := make(map[string]bool)
	for scanner.Scan() {
		if d := strings.TrimSpace(scanner.Text()); d != "" {
			blacklist[strings.ToLower(d)] = true
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Fehler beim Einlesen der Blacklist: %v", err)
	}
	log.Infof("Blacklist loaded with %d domains", len(blacklist))
	if len(blacklist) == 0 {
		log.Info("No blacklisted domains found, blacklist is empty")
	}
	// ende blacklist öffnen

	// 1) Template-Engine initialisieren
	engine := htmlTemplate.New("./views", ".html")
	// fiber-Template-Engine registrieren
	app := fiber.New(fiber.Config{
		Immutable: true,
		Views:     engine,
	})
	// Rate Limiting Middleware
	app.Use(limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool {
			return getrequestIP(c, config) == "127.0.0.1"
		},
		// Rate limit auf 30 Anfragen pro Minute pro IP
		Max:        30,
		Expiration: 120 * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Hier wird die IP-Adresse des Clients verwendet, um das Rate Limiting zu steuern
			return getrequestIP(c, config)
		},
		LimitReached: func(c *fiber.Ctx) error {
			// logge dass der Nutzer zu schnell ist
			log.Warnf("url: %s, Rate limit exceeded for IP: %s", c.Path(), getrequestIP(c, config))
			//return sendreply(c, fiber.StatusTooManyRequests, "too many requests, please try again later")
			return c.Status(fiber.StatusTooManyRequests).SendFile("./assets/toofast.html")
		},
	}))

	// add logging of files loaded by /assets
	app.Use(func(c *fiber.Ctx) error {
		log.Infof("url: %s, IP: %v", c.Path(), getrequestIP(c, config))
		return c.Next()
	})
	app.Static("/assets", "./assets")
	app.Post("/create", func(c *fiber.Ctx) error {
		// 1) JSON einlesen
		var req struct {
			Url   string `json:"url"`
			Slug  string `json:"slug"`
			Exp   string `json:"exp"`
			Title string `json:"title"`
			Desc  string `json:"desc"`
		}
		//allgemeine antwort felder validierung
		if err := c.BodyParser(&req); err != nil {
			log.Errorf("Fehler beim Parsen der Anfrage, Nutzer error: %v, fehlerhaftes json: %v", err, req)
			// Fehlercode Invalide JSON-Antwort zurückgeben
			return sendreply(c, fiber.StatusBadRequest, "invalid json")
		}
		//url validierung
		if req.Url == "" {
			return sendreply(c, fiber.StatusBadRequest, "The Destination URL cannot be empty!")
		}
		parsedURL, err := url.ParseRequestURI(req.Url)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			return sendreply(c, fiber.StatusBadRequest, "Invalid URL format")
		}
		host := strings.ToLower(strings.Split(parsedURL.Host, ":")[0])
		if blacklist[host] {
			return sendreply(c, fiber.StatusBadRequest, "Sorry, you cannot use this URL, please choose another one")
		}
		// self reference protection, if the config disallows it, check if the domain of the URL is the same as the configured domain, if so, return an error
		if !config["allow-self-reference"].(bool) {
			if host == config["domain"].(string) {
				return sendreply(c, fiber.StatusBadRequest, "Sorry, you cannot use this URL, please choose another one")
			}
		}

		//validate slug
		if req.Slug == "" {
			//check for existing slug and generate slug
			maxsluglen := config["defaultgeneratedsluglength"].(int)
			slugAllowedCharacters := config["slug-allowed-characters"].(string)
			//generate slug with the length of maxsluglen
			slug, err := generateSlug(slugAllowedCharacters, maxsluglen, db)
			if err != nil {
				log.Fatalf("Fehler beim Generieren des Slugs: %v", err)
				sendreply(c, fiber.StatusInternalServerError, "Internal error (db)")
			}
			if slug == "" {
				return sendreply(c, fiber.StatusInternalServerError, "Internal error (no unique slug found)")
			}
			req.Slug = slug
		} else {
			if !slugRegex.MatchString(req.Slug) {
				return sendreply(c, fiber.StatusBadRequest, "Slug darf nur Buchstaben, Zahlen, Unterstriche und Bindestriche enthalten")
			}
			//check if slug already exists in db
			var (
				existingSlug string
			)
			err := db.QueryRow(`
					SELECT slug
					FROM links
					WHERE slug = ?
					LIMIT 1
				`, req.Slug).Scan(&existingSlug)
			if err == sql.ErrNoRows {
				// slug is unique
			} else if err != nil {
				log.Errorf("Datenbank Fehler: %v", err)
				return sendreply(c, fiber.StatusInternalServerError, "Internal error (db)")
			} else {
				// slug already exists
				return sendreply(c, fiber.StatusBadRequest, "This slug is already in use, please choose another one")
			}
		}
		//validate expiration date
		if req.Exp != "" {
			expDate, err := time.Parse("2006-01-02", req.Exp)
			if err != nil {
				return sendreply(c, fiber.StatusBadRequest, "invalid date format")
			}
			today := time.Now().Truncate(24 * time.Hour)
			if expDate.Before(today) {
				return sendreply(c, fiber.StatusBadRequest, "expiration date cannot be earlier than today")
			}
		}
		//validate title
		req.Title = strings.TrimSpace(req.Title)
		if !titleRe.MatchString(req.Title) {
			return sendreply(c, fiber.StatusBadRequest, "Titel ungültig (Länge bis 100, keine ‹› oder Steuerzeichen)")
		}
		//validate desc
		req.Desc = strings.Trim(req.Desc, " 	")
		if !descRe.MatchString(req.Desc) {
			return sendreply(c, fiber.StatusBadRequest, "Beschreibung ungültig (Länge bis 1000, keine ‹› oder Steuerzeichen)")
		}
		//write values to db in a prepared statement, url var is parsedurl, client IP is the return value of the function getrequestIP(c, config)
		//insert values into db
		// Prepared Statement zum Einfügen
		stmt, err := db.Prepare(`
            INSERT INTO links (url, slug, exp, title, desc, created_ip)
            VALUES (?, ?, ?, ?, ?, ?)
        `)
		if err != nil {
			log.Errorf("Fehler beim Vorbereiten der SQL-Anweisung: %v", err)
			return sendreply(c, fiber.StatusInternalServerError, "Internal Server Error (db)")
		}
		defer stmt.Close()

		// Werte aus req + Client-IP übergeben
		if _, err := stmt.Exec(
			req.Url,
			req.Slug,
			req.Exp,
			req.Title,
			req.Desc,
			getrequestIP(c, config), // Client-IP
		); err != nil {
			log.Errorf("Fehler beim Einfügen in die Datenbank: %v", err)
			return sendreply(c, fiber.StatusInternalServerError, "Internal Server Error (db)")
		}
		domain := config["domain"].(string)
		https := config["uses-https"].(bool)
		if domain == "" {
			// Fallback auf localhost, wenn keine Domain angegeben ist
			domain = "localhost"
		}
		// url für die Antwort generieren
		finalurl := "http" + "://" + domain + "/" + req.Slug
		if https {
			finalurl = "https://" + domain + "/" + req.Slug
		}
		return sendreply(c, fiber.StatusOK, finalurl)

		//v := req.Url + "\n" + req.Slug + "\n" + req.Exp + "\n" + req.Title + "\n" + req.Desc
	})
	app.Static("/", "./assets/index.html")
	app.Static("/favicon.ico", "./assets/j44-64x64.ico")
	app.Get("/:slug?", func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		if !slugRegex.MatchString(slug) {
			// 400 Bad Request bei ungültigem Format
			return sendreply(c, fiber.StatusBadRequest, "Invalid Request")
		}
		safeSlug := html.EscapeString(slug)
		//hier sqlite db abfragen nach daten
		var (
			url, title, desc, expiration string
		)
		err := db.QueryRow(`
					SELECT url, title, desc, exp
					FROM links
					WHERE slug = ?
					LIMIT 1
				`, safeSlug).Scan(&url, &title, &desc, &expiration)
		if err == sql.ErrNoRows {
			// 404 Template rendern
			return c.Render("err", fiber.Map{
				"code":    "404",
				"message": "Not found",
			})
		} else if err != nil {
			log.Errorf("Datenbank Fehler: %v", err)
			return sendreply(c, fiber.StatusInternalServerError, "Internal error (db)")
		}

		dbdata := fiber.Map{
			"destination": url,
			"title":       title,
			"description": desc,
			"expiration":  expiration,
		}
		// prüfe ob url leer ist
		if dbdata["destination"] == "" {
			// 404 Template rendern
			return c.Render("err", fiber.Map{
				"code":    "404",
				"message": "Not found",
			})
		}
		//prüfe ob expiration leer ist
		if dbdata["expiration"].(string) != "" {
			//prüfe ob expiration in der zukunft liegt, falls nicht dann sende 410 Gone
			dbexdate, err := time.Parse("2006-01-02", dbdata["expiration"].(string))
			if err != nil {
				return sendreply(c, fiber.StatusBadRequest, "Invalid expiration date format")
			}
			if dbexdate.Before(time.Now()) {
				log.Infof("Page hit for: %s, but is expired :(", dbdata["slug"])
				// 410 Gone Template rendern
				return c.Render("err", fiber.Map{
					"code":    "410",
					"message": "Link expired",
				})
			} else {
				log.Infof("Page hit for: %s, redirecting to: %v", dbdata["slug"], dbdata["destination"])
			}
		}

		// Template rendern
		return c.Render("redirect", dbdata)
	})
	app.Listen(":3000")
}

// sendreply bereitet eine JSON-Error-Antwort vor und bricht den Handler ab.
func sendreply(c *fiber.Ctx, status int, message string) error {
	if status == fiber.StatusOK {
		return c.Status(status).JSON(fiber.Map{
			"finalurl": message,
		})
	}
	return c.Status(status).JSON(fiber.Map{
		"error": message,
	})
}
func initDB(dbPath string) (*sql.DB, error) {
	// Datenbank öffnen (oder neue Datei anlegen)
	// Treibername "sqlite", dbPath z.B. "file:links.db?cache=shared&mode=rwc"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	// SQL-Statement zur Tabellenerstellung
	createTable := `
    CREATE TABLE IF NOT EXISTS links (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
        url         TEXT NOT NULL,
        slug        TEXT NOT NULL UNIQUE,
        exp         TEXT,
        title       TEXT,
        desc        TEXT,
        created_ip  TEXT,
		created     TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
    );`
	// Tabelle anlegen
	if _, err := db.Exec(createTable); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// loadConfig lädt die Konfigurationsdatei und gibt sie als map zurück
func loadConfig(filePath string) (map[string]interface{}, error) {
	// defaultYAML enthält die Konfiguration, die in eine Datei geschrieben werden kann.
	const defaultYAML = `# This file is used to configure the application settings.
# the minimum length of a slug, if not specified, defaults to 2 characters, if all available slugs are used up, another character is added
defaultgeneratedsluglength: 2
#the domain name for the application, used for generating URLs, defaults to localhost
domain: localhost
# whether to use https in the url that is being copied to the clipboard, defaults to true
uses-https: false
# self reference in target urls can create infinite redirect loops, not recommended; default: false
allow-self-reference: false
# whether to use the blacklist.txt of disallowed URLs, defaults to true
use-url-blacklist: true
# the characters that are allowed in slugs, defaults to lowercase letters and numbers, without uppercase letters and 0, defaults to "abcdefghijklmnopqrstuvwxyz123456789"
slug-allowed-characters: "abcdefghijklmnopqrstuvwxyz123456789"
# use the X-Forwarded-For header to get the real IP address of the client, defaults to true, should be set to false if the application is not behind a reverse proxy
use-x-forwarded-for: true`
	// falls die datei nicht existiert, wird sie erstellt
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Warnf("Config file not found, creating a new one with default settings.")
		if err := os.WriteFile(filePath, []byte(defaultYAML), 0777); err != nil {
			return nil, err
		}
	}
	// Konfigurationsdatei einlesen
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	// YAML in eine map einlesen
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// generateSlug erstellt einen Slug aus dem Titel mit der angegebenen maximalen Länge
func generateSlug(allowedchars string, slugLength int, db *sql.DB) (string, error) {
	// wähle maxlength zufällige Zeichen aus dem erlaubten Zeichensatz
	var preslug strings.Builder
	// repeat code until we have a unique slug
	for e := 0; e < 100; e++ {
		for i := 0; i < 100; i++ {
			for i := 0; i < slugLength; i++ {
				preslug.WriteByte(allowedchars[rand.Intn(len(allowedchars))])
			}
			slugString := preslug.String()
			// check if slug already exists in db, if so, generate a new one, retry for a maximum of 100 times after that generate a new slug with one more character, do this repeatedly for max 100 times
			// idea: housekeeping task to create an array of free slugs

			var (
				id, slug string
			)
			err := db.QueryRow(`
					SELECT id, slug
					FROM links
					WHERE slug = ?
					LIMIT 1
				`, slugString).Scan(&id, &slug)
			if err == sql.ErrNoRows {
				// Slug ist eindeutig, also zurückgeben
				log.Infof("Generated slug: %s", slugString)
				return slugString, nil
			} else if err != nil {
				log.Errorf("Datenbank Fehler: %v", err)
				return slugString, err
			} else {
				// Slug existiert bereits, also erneut generieren
				slugLength++
				preslug.Reset() // Slug existiert bereits, also erneut generieren
			}
		}
	} // Wenn nach 100 Versuchen kein eindeutiger Slug gefunden wurde, Fehler zurückgeben
	log.Errorf("No unique slug found after 100 attempts, returning error")
	return "", nil
}
func getrequestIP(c *fiber.Ctx, config map[string]interface{}) string {
	// check if the X-Forwarded-For header is set, if not, use c.IP()
	var realIP string
	if config["use-x-forwarded-for"] == true {
		realIP = c.Get("X-Forwarded-For")
	}
	if realIP == "" {
		realIP = c.IP()
	}
	return realIP
}