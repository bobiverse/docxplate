(__AI generated__)

# Docxplate Placeholder Triggers

Docxplate supports adding special **triggers** and **commands** inside placeholders to dynamically alter document contents. You can use them to remove rows, cells, tables, etc. based on a placeholder’s value. Below are some examples and explanations that you can share with end users.

## Basic Placeholder
```
{{Customer.Name}}
```

**No triggers**: this simply inserts the value of `Customer.Name` (if provided).

## Placeholder With Trigger

You can append one or more directives after your placeholder name in the format:

```
{{Key :On:Command:Scope}}
```

- **On** — The condition to check (`:empty`, `:unknown`, `:=`).
- **Command** — The action to take if the condition matches (`:remove`, `:clear`).
- **Scope** — The part of the document to affect (`:placeholder`, `:cell`, `:row`, `:list`, `:table`, `:section`).

For example:
```
{{Customer.Name :empty:remove:row}}
```

This means:
> **Remove** the **row** if `Customer.Name` is **empty**.

### Common Triggers

| Trigger       | Meaning                                                              |
|---------------|----------------------------------------------------------------------|
| `:empty`      | Trigger if the value is empty or missing.                            |
| `:unknown`    | Trigger if the value is not recognized (not provided or not defined).|
| `:=`          | Trigger when the value is set (you can compare or check a condition).|

### Common Commands

| Command    | Meaning                                                                     |
|------------|-----------------------------------------------------------------------------|
| `:remove`  | Remove the entire scope (e.g., remove the row, cell, list, etc.).           |
| `:clear`   | Clear the placeholder's text, leaving the rest of the document structure.   |

### Common Scopes

| Scope            | Meaning                                      |
|------------------|----------------------------------------------|
| `:placeholder`   | Only affect the placeholder text itself.     |
| `:cell`          | Affect the cell (if in a table).             |
| `:row`           | Affect the entire row (if in a table).       |
| `:list`          | Affect the entire list (if in a list).       |
| `:table`         | Affect the entire table (if in a table).     |
| `:section`       | Affect the entire section (e.g., table, list)|

## Examples

Below are some quick examples of how to use triggers:

1. **Remove the row if empty**  
```
{{Customer.Name :empty:remove:row}}
```
- If `Customer.Name` has no value, the entire row is removed.

2. **Remove the table if empty**  
```
{{Order.Details :empty:remove:table}}
```

- If `Order.Details` is empty, the entire table is removed.

3. **Clear the placeholder text if empty**  
```
{{Notes :empty:clear:placeholder}}
```
- If `Notes` is empty, the placeholder text disappears but the cell/table remains.

4. **Remove an entire list if empty**  
```
{{Items :empty:remove:list}}
```
- If `Items` has no value, the entire list is removed.

5. **Remove the cell if empty**  
```
{{Customer.ID :empty:remove:cell}}
```
- If `Customer.ID` is empty, the cell containing this placeholder is removed.

## Summary

- If you **don’t** need any special behavior, just use `{{Placeholder}}`.
- Add `:On:Command:Scope` when you want docxplate to perform an action based on whether the placeholder value is empty, unknown, or a certain value.

That’s it! With these simple placeholders and triggers, your docxplate users can control whether to remove rows, clear placeholders, or remove entire tables based on the value of a field.
