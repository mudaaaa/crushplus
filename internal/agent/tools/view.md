Reads and displays file contents with line numbers for examining code, logs, or text data.

<usage>
- Provide full file path to read (REQUIRED, string)
- Optional offset: start reading from specific line (0-based, INTEGER)
- Optional limit: control lines read (default 2000, INTEGER)
- Don't use for directories (use LS tool instead)
</usage>

<parameters>
- file_path (string, required): The path to the file to read
- offset (integer, optional): The line number to start reading from (0-based)
- limit (integer, optional): The number of lines to read (defaults to 2000)
</parameters>

<features>
- Displays contents with line numbers
- Can read from any file position using offset
- Handles large files by limiting lines read
- Auto-truncates very long lines for display
- Suggests similar filenames when file not found
</features>

<limitations>
- Max file size: 250KB
- Default limit: 2000 lines
- Lines >2000 chars truncated
- Cannot display binary files/images (identifies them)
</limitations>

<cross_platform>
- Handles Windows (CRLF) and Unix (LF) line endings
- Works with forward slashes (/) and backslashes (\)
- Auto-detects text encoding for common formats
</cross_platform>

<tips>
- Use with Glob to find files first
- For code exploration: Grep to find relevant files, then View to examine
- For large files: use offset parameter for specific sections
- IMPORTANT: offset and limit must be integers, not strings (e.g., use offset=100, not offset="100")
- When using the tool programmatically, ensure numeric parameters are passed as numbers, not quoted strings
</tips>
