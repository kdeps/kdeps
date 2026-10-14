# Messaging resources

The email and twiml components and the bot reply for reaching a person. Each has its own reference page.

Works in workflow and agent mode.

| Resource | Channel | Reference |
| :--- | :--- | :--- |
| `component: email` | SMTP send, IMAP read / search / modify | [Email](/workflow/resources/email) |
| `component: twiml` | Voice calls - TwiML say, gather, dial, record for a phone webhook | [Telephony](/workflow/resources/telephony) |
| `apiResponse:` (bot mode) | The workflow result is sent to the chat that delivered the message | [Bot replies](/workflow/resources/bot-reply) |

## See also

- [Phone assistant tutorial](/examples/telephony-bot) - phone webhooks step by step
- [Stateless bot](/examples/stateless-bot/) and [Telegram bot](/examples/telegram-bot/) - bot replies in full
- [Global config](/reference/advanced-config) - where SMTP, IMAP, and bot credentials live
- [Input sources](/workflow/input-sources) - configuring `settings.input.bot`
- [Resources overview](/workflow/resources) - all resource types
