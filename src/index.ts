import dotenv from "dotenv";
import { createSolanaClient } from "./lib/solana";
import { initBot, setupStartCommand, setupStatusCommand, setupThresholdCommand } from "./lib/bot";
import { botConfig } from "./config";
import { startMonitor } from "./monitor";


async function main(): Promise<void> {
    // load env
    dotenv.config();
    const TOKEN = process.env.TELEGRAM_BOT_TOKEN;
    const CHAT_ID = process.env.CHAT_ID;
    const HELIUS_KEY = process.env.HELIUS_API_KEY;


    if (!TOKEN || !CHAT_ID || !HELIUS_KEY) {
        throw new Error("Missing required env variables");
    } else {
        botConfig.adminChatID = CHAT_ID
    }

    const solanaClient = createSolanaClient(HELIUS_KEY);

    const bot = initBot(TOKEN);
    startMonitor(solanaClient, bot);
    setupStartCommand(bot);
    setupStatusCommand(bot);
    setupThresholdCommand(bot);
    bot.launch();

    // Enable graceful stop
    process.once('SIGINT', () => bot.stop('SIGINT'))
    process.once('SIGTERM', () => bot.stop('SIGTERM'))
}

main();


