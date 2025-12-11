import { neon } from "@neondatabase/serverless";
import type {
  ScheduledController,
  ExecutionContext,
  R2Bucket,
} from "@cloudflare/workers-types";

// Define environment vars with correct types
interface Env {
  DB_URL: string;
  R2: R2Bucket;
}

export default {
  async scheduled(
    event: ScheduledController,
    env: Env,
    ctx: ExecutionContext,
  ): Promise<void> {
    const sql = neon(env.DB_URL);

    console.log("[Cron] Cleanup job started");

    // 1. Fetch expired transfers
    const expiredTransfers = await sql`
      SELECT id FROM transfers
      WHERE expires_at <= NOW() AND deleted = false;
    `;

    if (expiredTransfers.length === 0) {
      console.log("[Cron] No expired transfers found.");
      return;
    }

    console.log(`[Cron] Found ${expiredTransfers.length} expired transfers.`);

    for (const row of expiredTransfers) {
      const transferId = row.id;

      try {
        // 2. Fetch files for this transfer
        const files = await sql`
          SELECT path FROM files
          WHERE transfer_id = ${transferId} AND deleted = false;
        `;

        // 3. Delete each file from R2
        for (const f of files) {
          const key = f.path;
          try {
            await env.R2.delete(key);
            console.log(`Deleted R2 object: ${key}`);
          } catch (err) {
            console.error(`Failed to delete R2 object ${key}`, err);
          }
        }

        // 4. Update DB to mark deleted
        await sql`
          UPDATE transfers 
          SET deleted = true, updated_at = NOW()
          WHERE id = ${transferId};
        `;

        await sql`
          UPDATE files
          SET deleted = true, updated_at = NOW()
          WHERE transfer_id = ${transferId};
        `;

        console.log(`Transfer ${transferId} cleaned.`);
      } catch (error) {
        console.error(`Error cleaning transfer ${transferId}:`, error);
      }
    }

    console.log("[Cron] Cleanup job finished.");
  },
};
