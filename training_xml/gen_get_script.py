#! /usr/bin/python

import time
import sys
import xml.etree.ElementTree as ET
import requests


## This is not "producton" code. It is a utility to pull a ton
## of XML data from a BigFix server to use for "training XML"

bfsession = requests.Session
bfurlbase = "https://10.10.220.60:52311"
bfuser = "IEMAdmin"
bfpass = "BigFix!123"
bfsess = requests.Session()
bfsess.auth = (bfuser, bfpass)
sm = {}
resp = bfsess.get(bfurlbase + "/api/login", verify=False)
if not resp.ok:
    print(f"Login failed for user {bfuser}")
    sys.exit(1)

initial_urls = [
    "/api/actions",
    "/api/computers",
    "/api/dashboardvariables",
    "/api/ldapdirectories",
    "/api/operators",
    "/api/properties",
    "/api/roles",
    "/api/samlproviders",
    "/api/sites",
    "/api/admin/fields",
    "/api/admin/masthead/parameters",
    "/api/admin/options",
    "/api/webreports",
]

def get_url(bfurl):
    """
    This function takes a url as an argument and returns the response.
    """
    req = requests.Request("GET", bfurl)
    res = bfsess.send(bfsess.prepare_request(req))

    if not res.ok:
        print(f"Error: {res.status_code} Reason: {res.reason}")
        return None

    return res.text


def process_url(root):
    """
    This function processes the urls in the root.
    """
    for obj in root:
        print(obj)
        objtag = obj.tag
        objurl = obj.get("Resource")
        # TODO: Do something with objtag and objurl
        print(f"Type {objtag} URL {objurl}")

        if objurl is not None:
            besfilename = objurl.split("/api/")[1].replace("/", "_")
        else:
            besfilename = objtag

        if objtag == "DashboardData":
            dvurl = objurl
            dvresponse = get_url(dvurl)
            dvroot = ET.fromstring(dvresponse)
            dvpathanme = f"besapi_DBV_{besfilename}_V.xml"
            if len(dvpathanme) > 255:
                print(f"Filename {dvpathanme} is too long")
                continue

            with open(dvpathanme, "w", encoding="utf-8") as f:
                f.write(dvresponse)

        if objtag == "Action":
            actid = objurl.split("/")[-1]
            resurl = objurl + "/status"
            resresponse = get_url(resurl)
            with open(f"besapi_action_{actid}_result.xml", "w", encoding="utf-8") as f:
                f.write(resresponse)

        elif objtag.endswith("Site"):
            sm[objtag] = sm.get(objtag,0) + 1
            if sm[objtag] > 2:
                continue
            siteid = objurl.split("/")[-1]
            conturl = objurl + "/content"
            contresponse = get_url(conturl)
            controot = ET.fromstring(contresponse)
            with open(f"besapi_site_{siteid}_content.xml", "w", encoding="utf-8") as f:
                f.write(contresponse)
            process_url(controot)

        lfn = "TBD"

        with open("get_training_xml.sh", "a", encoding="utf-8") as f:
            f.write(f"curl --insecure -u '{bfuser}:{bfpass}' '{objurl}' " +
                    f"-o '{lfn}_{besfilename}.xml'\n")




def main():
    """
    This function is the entry point of the script.
    """
    with open("get_training_xml.sh", "w", encoding="utf-8") as f:
        f.write("#! /bin/bash\n")
    start_time = time.time()
    for url in initial_urls:
        print(f"Processing {url}...")
        objname = url.rsplit("/", maxsplit=1)[-1]
        response = get_url(bfurlbase + url)
        with open(f"besapi_{objname}.xml", "w", encoding="utf-8") as f:
            f.write(response)

        root = ET.fromstring(response)
        process_url(root)

    print("Processing complete.")
    print(f"Total elapsed time: {time.time() - start_time:.2f} seconds")
    with open("get_training_xml.sh", "a", encoding="utf-8") as f:
        f.write("echo 'Run completed'\n")


if __name__ == "__main__":
    main()
