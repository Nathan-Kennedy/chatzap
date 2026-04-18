package service

import "strings"

// stripWhatsAppUserDeviceSuffix remove ":device" do user JID (multi-dispositivo / Business).
// Ex.: 5569993378283:45@s.whatsapp.net → 5569993378283 (antes de normalizar dígitos).
// Sem isto, digitsOnly fundia dígitos do device ao número e quebrava o match na BD.
func stripWhatsAppUserDeviceSuffix(local, domain string) string {
	local = strings.TrimSpace(local)
	d := strings.ToLower(strings.TrimSpace(domain))
	if strings.Contains(d, "g.us") || d == "lid" {
		return local
	}
	if d != "s.whatsapp.net" && d != "c.us" {
		return local
	}
	if i := strings.IndexByte(local, ':'); i > 0 {
		return local[:i]
	}
	return local
}

// NormalizeContactJID converte telefone ou JID para chave canónica na BD (evita conversas duplicadas).
// Chats individuais: só dígitos internacionais + @s.whatsapp.net.
// Grupos: preserva identificador + @g.us.
func NormalizeContactJID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "@") {
		parts := strings.SplitN(s, "@", 2)
		local := strings.TrimSpace(parts[0])
		domain := strings.ToLower(strings.TrimSpace(parts[1]))
		if strings.Contains(domain, "g.us") {
			return local + "@" + domain
		}
		local = stripWhatsAppUserDeviceSuffix(local, domain)
		// LID do WhatsApp: não converter para @s.whatsapp.net (quebrava o match com webhooks).
		if domain == "lid" {
			d := digitsOnly(local)
			if d == "" {
				return strings.ToLower(strings.TrimSpace(s))
			}
			return d + "@lid"
		}
		d := digitsOnly(local)
		if d == "" {
			return strings.ToLower(strings.TrimSpace(s))
		}
		d = brMobileCanonicalNational(d)
		return d + "@s.whatsapp.net"
	}
	d := digitsOnly(s)
	d = applyDefaultBRCountryCode(d)
	if d == "" {
		return ""
	}
	d = brMobileCanonicalNational(d)
	return d + "@s.whatsapp.net"
}

// brMobileCanonicalNational: móvel BR 55+DDD+8 dígitos começados por 9 → insere o 9 nacional após o DDD (13 dígitos).
// Ex.: 556993378283 (falta um 9) → 5569993378283. Linhas fixas com 8 dígitos sem 9 inicial não alteram.
func brMobileCanonicalNational(d string) string {
	if len(d) != 12 || !strings.HasPrefix(d, "55") {
		return d
	}
	rest := d[4:]
	if len(rest) != 8 || rest[0] != '9' {
		return d
	}
	return d[:4] + "9" + rest
}

// brMobileCollapsedNational: inverso canónico (13 → 12) para encontrar conversas antigas gravadas sem o 9 extra.
func brMobileCollapsedNational(d string) string {
	if len(d) != 13 || !strings.HasPrefix(d, "55") {
		return ""
	}
	rest := d[4:]
	if len(rest) != 9 || rest[0] != '9' {
		return ""
	}
	return d[:4] + rest[1:]
}

// expandBRWhatsAppJIDs devolve JIDs a usar na procura (canónico primeiro, depois variante 12 dígitos).
func expandBRWhatsAppJIDs(jid string) []string {
	if !strings.HasSuffix(jid, "@s.whatsapp.net") {
		return []string{jid}
	}
	d := digitsOnly(strings.TrimSuffix(jid, "@s.whatsapp.net"))
	if len(d) < 10 || !strings.HasPrefix(d, "55") {
		return []string{jid}
	}
	canon := brMobileCanonicalNational(d)
	cjid := canon + "@s.whatsapp.net"
	var out []string
	seen := make(map[string]struct{})
	push := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	push(cjid)
	if coll := brMobileCollapsedNational(canon); coll != "" && coll != canon {
		push(coll + "@s.whatsapp.net")
	}
	return out
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// applyDefaultBRCountryCode: números BR sem +55 (10 ou 11 dígitos) recebem prefixo 55.
// DisplayNameFromJID nome simples derivado do JID (fallback quando o webhook não envia nome).
func DisplayNameFromJID(jid string) string {
	s := strings.Split(jid, "@")[0]
	s = strings.ReplaceAll(s, ".", " ")
	return s
}

func applyDefaultBRCountryCode(digits string) string {
	if digits == "" {
		return ""
	}
	if len(digits) >= 12 && strings.HasPrefix(digits, "55") {
		return digits
	}
	if len(digits) == 10 || len(digits) == 11 {
		return "55" + digits
	}
	return digits
}

// CollectJIDLookupKeys devolve candidatos à mesma conversa (PN primeiro quando existe remoteJidAlt).
// Inclui variantes BR 12/13 dígitos para não duplicar conversas (webhook vs operador).
func CollectJIDLookupKeys(remoteJid, remoteJidAlt string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		j := NormalizeContactJID(raw)
		if j == "" {
			return
		}
		for _, variant := range expandBRWhatsAppJIDs(j) {
			if _, ok := seen[variant]; ok {
				continue
			}
			seen[variant] = struct{}{}
			out = append(out, variant)
		}
	}
	add(remoteJidAlt)
	add(remoteJid)
	return out
}

// InboundCanonicalJID chave preferida (PN) para nova conversa ou envio; igual ao primeiro candidato de CollectJIDLookupKeys.
func InboundCanonicalJID(remoteJid, remoteJidAlt string) string {
	keys := CollectJIDLookupKeys(remoteJid, remoteJidAlt)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}
