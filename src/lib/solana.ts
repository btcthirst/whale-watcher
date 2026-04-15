import { Connection } from "@solana/web3.js";

export function createSolanaClient(heliusKey: string) {
    const rpcUrl = `https://devnet.helius-rpc.com/?api-key=${heliusKey}`;
    return new Connection(rpcUrl, "confirmed");
}