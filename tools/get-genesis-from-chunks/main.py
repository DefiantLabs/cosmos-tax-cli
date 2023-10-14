import argparse
import requests
import json
import sys
import time
import base64

parser = argparse.ArgumentParser(description='Get genesis from chunks')

parser.add_argument('-n', '--node-url', type=str, required=True, help='Node url')
parser.add_argument('-ucc', '--use-cached-chunks', action="store_true", help='Node url')

def main():
    args = parser.parse_args()

    node_url = args.node_url
    use_cached_chunks = args.use_cached_chunks

    curr_chunk = 0
    total = 1
    total_set = False

    chunked_data = []
    if not use_cached_chunks:
        while True:
            curr_chunked_resp = requests.get(f'{node_url}/genesis_chunked?chunk={curr_chunk}')
            try:
                curr_chunked_resp.raise_for_status()
            except:
                print('Error getting chunked response')
                print(curr_chunked_resp.status_code)
                print(curr_chunked_resp.text)
                sys.exit(1)

            print("Got chunk", curr_chunk)
            curr_chunked_data = curr_chunked_resp.json()["result"]

            if not total_set:
                total = int(curr_chunked_data['total'])
                total_set = True

            chunked_data.append(str(curr_chunked_data['data']))

            if curr_chunk == total - 1:
                break

            curr_chunk += 1

            time.sleep(2)

        json.dump(chunked_data, open("output/chunked_data.json", 'w'))
    else:
        chunked_data = json.load(open("output/chunked_data.json", 'r'))

    full_data = ""
    for data in chunked_data:
        decoded_chunk = base64.b64decode(data).decode('utf-8')
        full_data += decoded_chunk

    data_obj = json.loads(full_data)
    print(data_obj.keys())
    json.dump(data_obj, open("output/full_data.json", 'w'))

if __name__ == '__main__':
    main()
