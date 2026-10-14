# Messaging resources

Two resources and the bot reply for reaching a person. Each has its own reference page.

Works in workflow and agent mode.

| Resource | Channel | Reference |
| :--- | :--- | :--- |
| `email:` | SMTP send, IMAP read / search / modify | [Email](/workflow/resources/email) |
| `telephony:` | Voice calls - say, ask, menu, dial, record | [Telephony](/workflow/resources/telephony) |
| `apiResponse:` (bot mode) | The workflow result is sent to the chat that delivered the message | [Bot replies](/workflow/resources/bot-reply) |

## See also

- [Phone assistant tutorial](/examples/telephony-bot) - `telephony:` step by step
- [Stateless bot](/examples/stateless-bot/) and [Telegram bot](/examples/telegram-bot/) - bot replies in full
- [Global config](/reference/advanced-config) - where SMTP, IMAP, and bot credentials live
- [Input sources](/workflow/input-sources) - configuring `settings.input.bot`
- [Resources overview](/workflow/resources) - all resource types
