---
name: weather
description: How to answer questions about the weather using a public API.
---
To answer a weather question:

1. Determine the location the user is asking about.
2. Use the `http_get` tool to fetch `https://wttr.in/<location>?format=3`.
3. Summarize the result in one friendly sentence.

If the location is ambiguous, ask the user to clarify before fetching.
