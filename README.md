# Link Zipper

Сервис для скачивания публичных файлов по URL и упаковки их в ZIP-архив.

## Возможности

- Создание задачи на упаковку до 3 файлов (.pdf, .jpeg)
- Автоматическая сборка архива после добавления третьего файла
- Ограничение на 3 одновременные задачи
- Информирование об ошибках при недоступности ресурсов
  
## Паттерны и практики

* In-Memory TaskManager с `sync.Mutex`

## Конфигурация

Файл `config.yaml`:

```yaml
server:
  port: 8080
limits:
  maxTasks: 3
  maxFilesPerTask: 3
  allowedExtensions:
    - ".pdf"
    - ".jpeg"

logging:
  level: info
  file: server.log
```
