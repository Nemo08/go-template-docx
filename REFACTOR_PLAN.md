# План архитектурного рефакторинга go-template-docx

## ✅ Выполнено

### Шаг 1. SOLID-рефакторинг docxTemplate
- `docxTemplate` содержит `Config TemplateConfig` + `State TemplateState` (вместо 16 полей)
- `Apply()` передаёт их по указателю — без ручного копирования
- Удалены 8+ мёртвых методов с `docxTemplate`
- Тесты обновлены

### Шаг 2. Перенос `xml` → `internal/xml`
- Пакет `xml` перемещён в `internal/xml` — убран из публичного API
- `documentMeta` разделён на `documentConfig` + `documentState`

### Шаг 3. Удаление всех `init()`
- `template_funcs.go`: `validHighlightColor`/`namedStyles` — var-init closures; `extraFuncMap` влит в `TemplateFuncs`
- `preprocessors.go`: `DefaultPreProcessors` var → `func() []xml.HandlersMap`
- `postprocessors.go`: `DefaultPostProcessors` var → `func() []xml.HandlersMap`
- `hideRowPostProcessor`/`pageBreakPostProcessor` заинлайнены

### Шаг 4. Разбивка `modifyXlsxInMemory` (complexity 18 → 8)
- `copyNonMatchingXlsxFiles` — копирование non-sheet/non-ss файлов (complexity 3)
- `loadAndProcessSharedStrings` — чтение + шаблонизация + чистка shared strings (complexity 5)
- `processXlsxSheets` — цикл по листам с chartValues (complexity 7)

## 📋 Оставшиеся шаги

### Шаг 5. Разделение `internal/docx` на подпакеты
Текущий `internal/docx` — god-package (~570 символов, 25 файлов).

**Предлагаемое разделение:**

| Подпакет | Назначение | Файлы |
|---|---|---|
| `internal/docx/templates` | Функции шаблонов | `template_funcs.go`, `template_funcs_placeholders.go`, `apply_template.go`, `wrap_missing_keys.go` |
| `internal/docx/autoexpand` | AutoExpandRows | `autoexpand.go` |
| `internal/docx/images` | Изображения и чарты | `media.go`, `chart.go` |
| `internal/docx/rels` | OOXML relationships | `rel.go` |
| `internal/docx` (остаётся) | Ядро пакета | `paths.go`, `document.go`, `content_types.go`, `processing.go`, `xml.go`, `preprocessors.go`, `postprocessors.go`, `prefix.go` |

- `paths.go` — может быть выделен в `internal/docx/paths`, но лучше сохранить в ядре
- Все публичные типы и константы (если нужны снаружи `internal/docx`) должны остаться в корне пакета или быть экспортированы через реэкспорт

### Шаг 6. Проверка импортов
- Обновить все импорты в `go_template_docx.go`, `internal/xlsx/go_chart.go`, тестах
- Убедиться, что `internal/xml` нигде не импортируется извне `internal/`

### Шаг 7. Финальная проверка
- `go build ./...`
- `go vet ./...`
- `go test -count=1 ./...`
- `golangci-lint run`
