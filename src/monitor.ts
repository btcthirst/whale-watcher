import { Connection } from "@solana/web3.js";
import { botConfig } from "./config";
import { parseTransaction, TransactionInfo } from "./parser";
import { sendAlert } from "./lib/bot";
import { Telegraf } from "telegraf";

export async function startMonitor(conn: Connection, bot: Telegraf) {

    conn.onLogs("all", async ({ signature, err }) => {
        if (err) return;
        const info = await parseTransaction(conn, signature);
        if (!info) return;
        if (isBelowThreshold(info)) return;
        console.log("Whale detected:", info);
        await sendAlert(bot, messageCreator(info));
    }, "confirmed");
}

export function isBelowThreshold(tx: TransactionInfo): boolean {
    return (tx.amount < botConfig.solThreshold)
}

export function messageCreator(tx: TransactionInfo): string {
    return `Whale detected: ${tx.amount} SOL from ${tx.from} to ${tx.to}`
}