#! /bin/bash
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/computers > api_computers.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/computer/9883517 > api_computer_9883517.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/computer/13419998 > api_computer_13419998.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/actions > api_actions.xml
curl --insecure -u 'IEMADmin:BigFix!123' https://10.10.220.60:52311/api/action/1267 > api_action_1267.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/action/5340 > api_action_5340.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/action/5432 > api_action_5432.xml
