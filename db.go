package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var db *sql.DB

// InitDB инициализирует подключение к базе данных
func InitDB(connStr string) error {
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	// Пинг для проверки соединения
	if err := db.Ping(); err != nil {
		return err
	}
	log.Println("Подключение к базе данных установлено")
	return nil
}

// CloseDB закрывает соединение с базой данных
func CloseDB() {
	if db != nil {
		db.Close()
	}
}

// CreateUserSession создаёт новую сессию для пользователя (если ещё не создана)
func CreateUserSession(userID int64) {
	// Пример запроса для создания сессии, если её ещё нет
	_, err := db.Exec(`INSERT INTO sessions (user_id) 
		VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID)
	if err != nil {
		log.Printf("Ошибка создания сессии для пользователя %d: %v", userID, err)
	}
}

// UpdateUserSession обновляет данные сессии (например, выбранный источник или токены)
func UpdateUserSession(userID int64, key, value string) {
	_, err := db.Exec(`UPDATE sessions SET data = jsonb_set(data, '{`+key+`}', to_jsonb($1::text)) WHERE user_id = $2`, value, userID)
	if err != nil {
		log.Printf("Ошибка обновления сессии для пользователя %d: %v", userID, err)
	}
}

// GetUserSession возвращает карту сессии пользователя
func GetUserSession(userID int64) map[string]string {
	session := make(map[string]string)
	row := db.QueryRow(`SELECT data FROM sessions WHERE user_id = $1`, userID)
	var data string
	if err := row.Scan(&data); err != nil {
		log.Printf("Ошибка получения сессии для пользователя %d: %v", userID, err)
		return session
	}
	// Простейший разбор JSON (для демонстрации; на практике используйте encoding/json)
	// Ожидается формат: {"source":"spotify", "spotify_token":"..."}
	// Здесь просто заглушка
	session["source"] = "spotify" // Значение по умолчанию, замените реальным парсингом
	return session
}

// GetUserSessionValue возвращает конкретное значение из сессии
func GetUserSessionValue(userID int64, key string) string {
	session := GetUserSession(userID)
	return session[key]
}
