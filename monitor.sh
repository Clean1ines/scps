#!/bin/sh
# monitor.sh – Скрипт для мониторинга и автоматического перезапуска приложения

APP="./scps"
LOGFILE="app.log"

while true; do
    echo "$(date): Запуск приложения" >> $LOGFILE
    $APP >> $LOGFILE 2>&1
    echo "$(date): Приложение завершило работу. Перезапуск через 5 секунд..." >> $LOGFILE
    sleep 5
done