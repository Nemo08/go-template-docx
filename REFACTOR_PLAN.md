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

**✅ Выполнено:**
- `internal/docx/autoexpand` — `autoexpand.go` + `autoexpand_test.go`
- `internal/docx/rels` — `rel.go` + `rel_test.go` + `types.go` (MediaRel, ImageMediaType, XMLStdHeader)
- `internal/docx/images` — `chart.go` + `chart_test.go` (UpdateChart, ApplyTemplateToXML, ExtractChartFilename)
- `wrapMissingKeys` → `WrapMissingKeys` (экспортирован для images)

**❌ Заблокировано (остаётся в core):**

| Почему не вынесено | Причина |
|---|---|
| `media.go` (Media, MediaMap) | `computeDocxImageSize` — метод на `*documentMeta` из `document.go` |
| `template_funcs*.go` | `template_funcs_placeholders.go` — методы на `*documentMeta`; используют Media, MediaMap, XMLImageData |
| `apply_template.go` | методы на `*documentMeta`; использует WrapMissingKeys, PatchXML |
| `wrap_missing_keys.go` | `EscapeTemplateValues` используется внешне; `WrapMissingKeys` используется `apply_template.go` (core) и `images/chart.go` |

### Шаг 6. Проверка импортов
- Убедиться, что `internal/xml` нигде не импортируется извне `internal/`
- Проверить, что все импорты `internal/docx` из внешних проектов обновлены

### Шаг 7. Финальная проверка
- `go build ./...`
- `go vet ./...`
- `go test -count=1 ./...`
