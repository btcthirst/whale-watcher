import { Telegraf } from "telegraf";
import { botConfig } from "../config";

// 1. Функція ініціалізації
export function initBot(token: string) {
    return new Telegraf(token);
}

// 2. Команда Start
export function setupStartCommand(bot: Telegraf) {
    bot.start((ctx) => {
        ctx.reply(`Привіт! Я моніторю великі переміщення SOL.\nПоріг: ${botConfig.solThreshold} SOL`);
    });
}

// 3. Команда Status
export function setupStatusCommand(bot: Telegraf) {
    bot.command("status", (ctx) => {
        const status = botConfig.isMonitoringActive ? "активний ✅" : "вимкнений ❌";
        ctx.reply(`Моніторинг зараз: ${status}`);
    });
}

// 4. Команда Threshold
export function setupThresholdCommand(bot: Telegraf) {
    bot.command("threshold", (ctx) => {
        ctx.reply(`Поточний поріг: ${botConfig.solThreshold} SOL`);
    });
}

// 5. ТА САМА функція для сповіщень
export async function sendAlert(bot: Telegraf, message: string) {
    const chatID = botConfig.adminChatID || process.env.CHAT_ID;
    if (!chatID) {
        console.warn("[sendAlert] No chat ID - user must send /start first")
    } else {
        await bot.telegram.sendMessage(chatID, `⚠️ ALERT: ${message}`)
    }


}