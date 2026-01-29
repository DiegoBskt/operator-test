## 2025-05-02 - Unsafe HTML Generation
**Vulnerability:** Found `GenerateHTML` constructing reports using `fmt.Sprintf` with unescaped user input (cluster info, findings), leading to Stored XSS.
**Learning:** The project manually builds HTML reports instead of using `html/template`, making it prone to injection if developers forget to escape inputs.
**Prevention:** Use `html/template` for HTML generation or ensure all inputs are wrapped in `html.EscapeString` before concatenation.
