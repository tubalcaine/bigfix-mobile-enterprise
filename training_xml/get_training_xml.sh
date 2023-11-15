#! /bin/bash

# Get the core BESAPI components
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/computers > besapi_computers.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/actions > besapi_actions.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/fixlets/external/BES%20Support > besapi_fixlets.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/analyses/external/BES%20Support > besapi_analyses.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/sites > besapi_sites.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/baselines/custom/Utility%20Content > besapi_baselines.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/computergroups/master > besapi_computergroups.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/properties/master > besapi_properties.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/site/external/BES%20Support/content > besapi_sitecontent.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/tasks > besapi_tasks.xml

# Get the BES components
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/site/external/BES%20Support > bes_site.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/fixlet/external/BES%20Support/5 > bes_fixlet_5.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/fixlet/external/BES%20Support/75 > bes_fixlet_75.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/fixlet/external/BES%20Support/134 > bes_fixlet_134.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/task/external/BES%20Support/148 > bes_task_148.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/task/external/BES%20Support/154 > bes_task_154.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/task/external/BES%20Support/157 > bes_task_157.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/computer/541399441 > besapi_computer_541399441.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/computer/545757514 > besapi_computer_545757514.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/computer/546729256 > besapi_computer_546729256.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/action/3342 > bes_action_3342.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/action/1267 > bes_action_1267.xml
curl --insecure -u 'IEMAdmin:BigFix!123' https://10.10.220.60:52311/api/action/5374 > bes_action_5374.xml
