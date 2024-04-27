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

def get_chain_id_arg():
    parser = argparse.ArgumentParser(description="Dump failed block heights")
    parser.add_argument("--chain-id", type=str, default="osmosis-1", help="Chain ID to dump failed block heights from")
    args = parser.parse_args()
    return args.chain_id

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

if __name__ == "__main__":
    env = get_env()
    chain_id = get_chain_id_arg()
    os.makedirs("./output", exist_ok=True)

    conn = psycopg2.connect(f"dbname={env['db_name']} user={env['user']} host={env['host']} password={env['password']} port={env['port']}")

    try:

        rec = None
        with conn.cursor() as cur:
            cur.execute(SELECT_CHAINS_QUERY, (chain_id,))
            rec = cur.fetchone()
            if rec is None:
                raise Exception(f"Chain ID {chain_id} not found")

        with conn.cursor() as cur:
            cur.execute(GAPS_QUERY, (rec[0],))
            gaps = cur.fetchall()

        # flatten and fill the groups to get a list of missing heights
        flat = []
        for gap in gaps:
            flattened_and_filled = [x for x in range(gap[0], gap[1] + 1)]
            flat.extend(flattened_and_filled)

        json.dump(gaps, open("output/gaps.json", 'w'), indent=4)
        json.dump(flat, open("output/missing_heights.json", 'w'), indent=4)
    except Exception as err:
        print(err)
        traceback.print_exc()
    finally:
        conn.close()