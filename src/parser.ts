import { Connection, LAMPORTS_PER_SOL } from "@solana/web3.js";

export type TransactionInfo = {
    signature: string;
    from: string;
    to: string;
    amount: number;
    timestamp: number;
}

export async function parseTransaction(conn: Connection, signature: string): Promise<TransactionInfo | null> {
    const tx = await conn.getParsedTransaction(signature, {
        maxSupportedTransactionVersion: 0
    });

    if (!tx || !tx.meta) return null;

    const pre = tx.meta.preBalances;
    const post = tx.meta.postBalances;
    const keys = tx.transaction.message.accountKeys;

    let maxNeg = 0, maxPos = 0;
    let fromIdx = -1, toIdx = -1;

    keys.forEach((_, i) => {
        const change = post[i] - pre[i];
        if (change < maxNeg) { maxNeg = change; fromIdx = i; }
        if (change > maxPos) { maxPos = change; toIdx = i; }
    });

    if (fromIdx === -1 || toIdx === -1) return null;

    const amount = maxPos / LAMPORTS_PER_SOL;
    const from = keys[fromIdx].pubkey.toString();
    const to = keys[toIdx].pubkey.toString();
    const timestamp = tx.blockTime ?? Date.now() / 1000;

    return {
        signature,
        from,
        to,
        amount,
        timestamp
    }

}