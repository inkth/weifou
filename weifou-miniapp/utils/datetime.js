function pad(n) {
  return String(n).padStart(2, '0');
}

// ISO/Date → "MM月DD日 HH:mm"
function fmtDateTime(input) {
  if (!input) return '';
  const d = new Date(input);
  return `${d.getMonth() + 1}月${d.getDate()}日 ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// 今天 yyyy-mm-dd
function todayStr() {
  const d = new Date();
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// date(yyyy-mm-dd) + time(HH:mm) → ISO 字符串（本地时区）
function toISO(date, time) {
  const dt = new Date(`${date}T${time}:00`);
  return dt.toISOString();
}

module.exports = { fmtDateTime, todayStr, toISO, pad };
