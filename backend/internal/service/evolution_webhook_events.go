package service

// EvolutionWebhookDefaultEvents lista completa alinhada à Evolution API v2
// (https://doc.evolution-api.com/v2/api-reference/webhook/set — enum `events`).
// Mantém webhookByEvents=false na app para um único URL (/webhooks/whatsapp/:instance).
var EvolutionWebhookDefaultEvents = []string{
	"APPLICATION_STARTUP",
	"QRCODE_UPDATED",
	"MESSAGES_SET",
	"MESSAGES_UPSERT",
	"MESSAGES_UPDATE",
	"MESSAGES_DELETE",
	"SEND_MESSAGE",
	"CONTACTS_SET",
	"CONTACTS_UPSERT",
	"CONTACTS_UPDATE",
	"PRESENCE_UPDATE",
	"CHATS_SET",
	"CHATS_UPSERT",
	"CHATS_UPDATE",
	"CHATS_DELETE",
	"GROUPS_UPSERT",
	"GROUP_UPDATE",
	"GROUP_PARTICIPANTS_UPDATE",
	"CONNECTION_UPDATE",
	"CALL",
	"NEW_JWT_TOKEN",
	"TYPEBOT_START",
	"TYPEBOT_CHANGE_STATUS",
}
