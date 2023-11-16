import requests
import sys

## This is not "producton" code. It is a utility to pull a ton
## of XML data from a BigFix server to use for "training XML"

bfsession = requests.Session
bfurlbase = "https://10.10.220.60:52311"
bfuser = "IEMAdmin"
bfpass = "BigFix!123"
bfsess = requests.Session()
bfsess.auth = (bfuser, bfpass)
resp = bfsess.get(bfurlbase + "/api/login", verify=False)
if not resp.ok:
    print(f"Login failed for user {bfuser}")
    sys.exit(1)

initial_urls = [
    "/api/actions",
    "/api/analyses",
    "/api/auditlog",
    "/api/authenticate",
    "/api/baselines",
    "/api/clientquery",
    "/api/clientqueryresults",
    "/api/computergroups",
    "/api/computers",
    "/api/dashboardvariables",
    "/api/fixlets",
    "/api/ldapdirectories",
    "/api/operators",
    "/api/properties",
    "/api/query",
    "/api/roles",
    "/api/samlproviders",
    "/api/serverinfo",
    "/api/sites",
    "/api/tasks"
]

def get_url(bfurl):
    """
    This function takes a url as an argument and returns the response.
    """
    req = requests.Request("GET", bfurlbase + bfurl)
    res = bfsess.send(bfsess.prepare_request(req))

    if not res.ok:
        print(f"Error: {res.status_code} Reason: {res.reason}")
        return None

    return res.text

def main():
    """
    This function is the entry point of the script.
    """
    for url in initial_urls:
        response = get_url(url)
        # TODO: Process response

if __name__ == "__main__":
    main()
    

if __name__ == "__main__":
    main()
