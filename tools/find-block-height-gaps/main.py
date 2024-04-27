import psycopg2
import os
import json
import traceback
import argparse

def get_env():
    ret = {
        "host": os.environ.get("DB_HOST", ""),
        "password": os.environ.get("DB_PASSWORD", ""),
        "user": os.environ.get("DB_USER", ""),
        "port": os.environ.get("DB_PORT", ""),
        "db_name": os.environ.get("DB_NAME", "")
    }

    if any([ret[x] == "" for x in ret]):
        raise Exception("Must provide env vars")

    return ret

def get_args():
    parser = argparse.ArgumentParser(description="Finds and outputs block height gaps in a chain")
    parser.add_argument("--chain-id", type=str, default="osmosis-1", help="Chain ID to dump failed block heights from")
    parser.add_argument("--flatten-and-fill", "-ff", action="store_true", help="Flatten and fill the gaps and output to a file next to the gaps file")
    args = parser.parse_args()
    return args

SELECT_CHAINS_QUERY = "SELECT id FROM chains WHERE chain_id=%s;"
GAPS_QUERY = """
SELECT height + 1 AS gap_start,
       next_height - 1 AS gap_end
FROM (
  SELECT height,
         LEAD(height) OVER (ORDER BY height) AS next_height
  FROM blocks WHERE blockchain_id = %s
) nr
WHERE height + 1 <> next_height;
"""

def flatten_and_fill(gaps):
    flat = []
    for gap in gaps:
        flattened_and_filled = [x for x in range(gap[0], gap[1] + 1)]
        flat.extend(flattened_and_filled)
    return flat

if __name__ == "__main__":
    env = get_env()
    args = get_args()
    chain_id = args.chain_id
    ff = args.flatten_and_fill
    os.makedirs("./output", exist_ok=True)

    conn = psycopg2.connect(f"dbname={env['db_name']} user={env['user']} host={env['host']} password={env['password']} port={env['port']}")

    try:

        rec = None
        print(f"Finding chain ID {chain_id}...")
        with conn.cursor() as cur:
            cur.execute(SELECT_CHAINS_QUERY, (chain_id,))
            rec = cur.fetchone()
            if rec is None:
                raise Exception(f"Chain ID {chain_id} not found")

        print("Executing gap finder...")
        with conn.cursor() as cur:
            cur.execute(GAPS_QUERY, (rec[0],))
            gaps = cur.fetchall()


        json.dump(gaps, open("output/gaps.json", 'w'), indent=4)

        if ff and len(gaps) > 0:
            print("Found gaps, flattening and filling...")
            flat = flatten_and_fill(gaps)
            json.dump(flat, open("output/missing_heights.json", 'w'), indent=4)

        print("Done")
    except Exception as err:
        print(err)
        traceback.print_exc()
    finally:
        conn.close()
