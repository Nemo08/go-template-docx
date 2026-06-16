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
- `internal/docx/autoexpand` — `autoexpand.go` + `autoexpand_test.go` (чистый вынос, без internal cross-deps)

**❌ Заблокировано (требует дополнительного рефакторинга):**

| Подпакет | Блокировка |
|---|---|
| `internal/docx/rels` | `rel.go` использует `MediaRel` из `media.go`; `rel_test.go` использует `MediaRel` + `ImageMediaType` |
| `internal/docx/images` | `chart.go` использует `PatchXML` и `wrapMissingKeys` (core); `media.go` типы используются `template_funcs_placeholders.go` (методы на `*documentMeta`) |
| `internal/docx/templates` | `template_funcs_placeholders.go` — методы на `*documentMeta` из `document.go`; использует `Media`/`MediaMap`/`XMLImageData` |

**Ключевая проблема:** циклическая зависимость между core ↔ images/rels/templates из-за `template_funcs_placeholders.go` (методы на `*documentMeta`) и `chart.go` (использует `PatchXML`/`wrapMissingKeys` из core).

**Решение:** требуется предварительный шаг — вынести `MediaRel` и `ImageMediaType` в общий пакет (например, `rels/` или новый `types/`), а `wrapMissingKeys` — встроить в `chart.go` (единственный потребитель). Тогда:
1. `rels/` → base + `MediaRel`, `ImageMediaType` (core импортирует rels)
2. `images/` → `chart.go` (использует `xmlutil.PatchXML` напрямую, без core wrapper)
3. `media.go` → остаётся в core (не может быть вынесен из-за `template_funcs_placeholders.go`)

### Шаг 6. Проверка импортов
- Убедиться, что `internal/xml` нигде не импортируется извне `internal/`
- Проверить, что все импорты `internal/docx` из внешних проектов обновлены

### Шаг 7. Финальная проверка
- `go build ./...`
- `go vet ./...`
- `go test -count=1 ./...`
