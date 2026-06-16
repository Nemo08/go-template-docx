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

### Шаг 5. Разбивка `ParseDocumentMeta` (complexity 17 → 7)
- `parseDocPrIDs` — поиск docPr ID в XML (complexity 5)
- `parseMaxRId` — поиск макс. rId в rels (complexity 6)
- `parseMaxImageNumber` — поиск макс. номера изображения (complexity 4)

### Шаг 6. Разбивка `applyShapesBgFillColor` (complexity 11 → 4)
- `applySolidFill` — обработка solidFill (complexity 3)
- `applyVmlGradientFill` — обработка gradientFill (complexity 6)

### Шаг 7. Разбивка `processHeadersFootersDocument` (complexity 17 → 7)
- `processFileSequence` — универсальный циклический обработчик нумерованных файлов (complexity 7)

### Шаг 8. Разбивка `ProcessedOutput` (complexity 12 → 5)
- `processFileWithHandlers` — обработка одного файла через цепочку Handler (complexity 6)

### Шаг 9. Разбивка `formatStylesTags` (complexity 13 → 4)
- `dispatchStyleTag` — маршрутизация по типу стиля (complexity 5)
- `parseFontSizeStyle` — fontSize/fs (complexity 3)
- `parseColorStyle` — #hex (complexity 3)
- `parseShadingStyle` — bg: (complexity 3)
- `parseNamedStyle` — named (b/i/u/s) (complexity 3)

## 📋 Оставшиеся шаги

### Шаг 10. Разделение `internal/docx` на подпакеты

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

### Шаг 11. Снижение оставшихся complexity > 10 (необязательно)

Осталось 6 функций с HIGH, все — type-switch по AST/Go-типам. Разбиение даёт мало выгоды:

| Функция | Complexity | Тип | Причина HIGH |
|---------|-----------|-----|-------------|
| `extractFieldNamesRec` | 16 | type-switch | switch по 12+ нодам шаблона |
| `collectVariables` | 15 | type-switch | switch по нодам шаблона |
| `defaultVal` | 14 | type-switch | switch по 10+ Go-типам |
| `aggregateCol` | 13 | type-switch | switch по операциям (sum/avg/count) |
| `applyTemplatePipeline` | 13 | pipeline | 11 последовательных err-check-ов |
| `WrapMissingKeys` | 12 | type-switch | switch по 3 типам + range + if |

**Решение:** Оставить, т.к. каждый case тривиален (1-3 строки), разбиение не улучшит читаемость.

### Шаг 12. Проверка импортов
- Убедиться, что `internal/xml` нигде не импортируется извне `internal/`
- Проверить, что все импорты `internal/docx` из внешних проектов обновлены

### Шаг 13. Финальная проверка
- `go build ./...`
- `go vet ./...`
- `go test -count=1 ./...`
