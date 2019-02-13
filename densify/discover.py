import requests
import datetime
import time
import argparse
import string
import subprocess
import sys
import json

# Parsing inputted arguments from batch
parser = argparse.ArgumentParser()

parser.add_argument('--address', dest='prom_addr', default='',
                    help='Name of Prometheus server')
parser.add_argument('--port', dest='prom_port', default='',
                    help='Port of Prometheus server')
parser.add_argument('--history', dest='history', default='1',
                    help='Amount of time to go back for data collection works with the interval and intervalSize settings')
parser.add_argument('--file', dest='config_file', default='NA',
                    help='Config file to load settings from')
parser.add_argument('--timeout', dest='timeout', default='3600',
                    help='Timeout for querying Prometheus')
parser.add_argument('--collectMethod', dest='collection', default='kubernetes',
                    help='Data collection: kubernetes')
parser.add_argument('--mode', dest='mode', default='current',
                    help='What containers to collect ones running now or all the ones that ran in last days: current or all')
parser.add_argument('--protocol', dest='protocol', default='http',
                    help='Protocol to use to connect to Prometheus')
parser.add_argument('--sslCertVerify', dest='verify', default='True',
                    help='Verify SSL certificates or not can be True|False|Directory containing certificates')
parser.add_argument('--aggregator', dest='aggregator', default='max',
                    help='Which aggregator for the data collection of controllers to use max|avg|min')
parser.add_argument('--interval', dest='interval', default='days',
                    help='interval to use for data collection. Can be days, hours or minutes')
parser.add_argument('--intervalSize', dest='interval_size', default='1',
                    help='Interval size to be used for querying. eg. default of 1 with default interval of days queries 1 day at a time')
parser.add_argument('--debug', dest='debug', default='false',
                    help='enable debug logging')
args = parser.parse_args()

debug_log=open('./data/log.txt', 'w+')
		
# Function to collect the data from Prometheus. Takes in the metric to collect Note this is usually a complext string as it is the full query we want to run, The tag in the json we want to search for under data usually it is "result" and the query type either query or query_range
def metricCollect(metric,dataTag,query_type):
	# Builds the query and the calls prometheus.
	metric_temp = str(args.protocol) + '://' + str(args.prom_addr) + ':' + str(args.prom_port)  + '/api/v1/' + query_type + '?query=' + metric
	resp = requests.get(url=metric_temp, timeout=int(args.timeout))
	# If debug is on then will print out the query ran to the terminal and the debug log.
	if str(args.debug) == 'true':
		print(metric_temp)
		debug_log.write(metric_temp + '\n')
	# if the response is not 200 (ok) then it will print and log the call, status code, and message and exit the script.
	if resp.status_code != 200:
		print(metric_temp)
		print(resp.status_code)
		print(resp.text)
		debug_log.write(metric_temp + '\n')
		debug_log.write(str(resp.status_code) + '\n')
		debug_log.write(resp.text + '\n')
		sys.exit(1)
	#Formatting the response that is in json to be a dictionary. 
	data = resp.json()
	# Parsing just the part of the data we need from the response and returning it. 
	data2 = data['data'][dataTag]
	return data2

# Function called to collect data when using query_range. This will handle figuring out the details of the multiple calls to make. 
def multiDayCollect(query,dataTag,current_time):
	data2 = []
	# if we are running in current we will only collect containers running right now and just make 1 call.
	if args.mode == 'current':					
		data2 += metricCollect(query,dataTag,'query')
	# else we need to make multiple calls and build out the results from each. 
	else:
		#set count to how much history we want this way we get the oldest data first and work back to current. As we are wanting the latest status and sizes to be ones use so need the newest copy of it last. 
		count = int(args.history)
		while count > -1:
			#setting a second count we will use for end size as need to have an interval that will change in size depending what size was set in parameters by default will be 1 for 1 day. 
			count2 = count - int(args.interval_size)
			#Sets the start and end times based on which interval we are using as each one will affect the timestamps differently. 
			if args.interval == 'days':
				start = (current_time - datetime.timedelta(days=count)).strftime("%Y-%m-%dT00:00:00.000Z")
				end = (current_time - datetime.timedelta(days=count2)).strftime("%Y-%m-%dT23:00:00.000Z")
			elif args.interval == 'hours':
				start = (current_time - datetime.timedelta(hours=count)).strftime("%Y-%m-%dT%H:00:00.000Z")
				end = (current_time - datetime.timedelta(hours=count2)).strftime("%Y-%m-%dT%H:00:00.000Z")
			elif args.interval == 'minutes':
				start = (current_time - datetime.timedelta(minutes=count)).strftime("%Y-%m-%dT%H:%M:00.000Z")
				end = (current_time - datetime.timedelta(minutes=count2)).strftime("%Y-%m-%dT%H:%M:00.000Z")
			
			# Takes the metric passed in and adds the start and end time to the end of the query and calls the function metricCollect to get the data and adds it to the current data in the list.
			metric = query + '&start=' + start + '&end=' + end + '&step=5m'
			data2 += metricCollect(metric,dataTag,'query_range')

			count -= int(args.interval_size)
	return data2

# writes out the workload specified. takes in the list of data from one of the above functions, the systems dictionary which contains all the systems we want to report on. The output file name, property for the workload writing out and how we break down the system identifiers by controller\pod and container_name	
def writeWorkload(data2,systems,file,property,name1,name2,namespace):
	#Opens the file and writes out the header
	f=open('./data/' + file + '.csv', 'w+')
	f.write('cluster,namespace,pod,container,Datetime,' + property + '\n')
	#Loops through the data array checking to see if the system has both name identifiers in it or not. Then we check to see that the system exists in the list we are collecting data for. 
	for i in data2:
		if name2 in i['metric']:
			if name1 in i['metric']:
				if namespace in i['metric']:
					if i['metric'][name2] !='':
						if i['metric'][namespace] in systems:
							if i['metric'][name1] in systems[i['metric'][namespace]]:
								if i['metric'][name2] in systems[i['metric'][namespace]][i['metric'][name1]]:
									#Go through the different values which are the times and value of the metric. 
									for j in i['values']:
										# Setting up a variable to use as the default one we use for controllers is blank if it is a standalone pod so need to backfill this in those cases. 
										x = i['metric'][name1]
										if i['metric'][name1] == '<none>':
											x = systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['pod_name']
										#write out each individual metric. 
										f.write(str(args.prom_addr) + ',' + i['metric'][namespace] + ',' + x.replace(';','.') + ',' + i['metric'][name2].replace(':','.') + ',' + datetime.datetime.fromtimestamp(j[0]).strftime('%Y-%m-%d %H:%M:%S.%f')[:-3] + ',' + j[1] + '\n')
			elif 'pod_name' in i['metric']:
				if namespace in i['metric']:
					if i['metric'][name2] !='':
						if i['metric'][namespace] in systems:
							if i['metric']['pod_name'] in systems[i['metric'][namespace]]:
								if i['metric'][name2] in systems[i['metric'][namespace]][i['metric']['pod_name']]:
									#Go through the different values which are the times and value of the metric. 
									for j in i['values']:
										# Setting up a variable to use as the default one we use for controllers is blank if it is a standalone pod so need to backfill this in those cases. 
										x = i['metric']['pod_name']
										if i['metric']['pod_name'] == '<none>':
											x = systems[i['metric'][namespace]][i['metric']['pod_name']][i['metric'][name2]]['pod_name']
										#write out each individual metric. 
										f.write(str(args.prom_addr) + ',' + i['metric'][namespace] + ',' + x.replace(';','.') + ',' + i['metric'][name2].replace(':','.') + ',' + datetime.datetime.fromtimestamp(j[0]).strftime('%Y-%m-%d %H:%M:%S.%f')[:-3] + ',' + j[1] + '\n')
	f.close()

# Writes out the config file with all the systems and there settings
def writeConfig(systems,type):
	# Opens and writes the header file. 
	f=open('./data/config.csv', 'w+')
	#This if statement was going to be used for when doing nodes data collection though this was delayed which is why there is no else as we would call it twice but only want 1 header. 
	if type == 'CONTAINERS':
		f.write('cluster,namespace,pod,container,HW Total Memory,OS Name,HW Manufacturer,HW Model,HW Serial Number\n')
	# Look through all the pods and containers and writes out the config info.
	for n in systems:
		for i in systems[n]:
			if i != 'namespace_labels' and i != 'cpu_limit' and i != 'cpu_request' and i != 'mem_limit' and i != 'mem_request':
				for j in systems[n][i]:
					# Ignores items in the pods level for items that are attributes on the pod and not containers with details under them. 
					if j !='' and j != 'pod_info' and j != 'pod_labels' and j != 'owner_kind' and j != 'owner_name' and j != 'pod_name' and j != 'current_size' and j != 'creation_time':
						x = i
						if i == '<none>':
							x = systems[n][i][j]['pod_name']
						f.write(str(args.prom_addr) + ',' + n + ',' + x.replace(';','.') + ',' + j.replace(':','.') + ',' + str(systems[n][i][j]['memory']) + ',Linux,' + type + ',' + n + ',' + n + '\n')
	f.close()
		
# Writes out the attributes file
def writeAttributes(systems,type):
	f=open('./data/attributes.csv', 'w+')
	if type == 'container':
		f.write('cluster,namespace,pod,container,Virtual Technology,Virtual Domain,Virtual Datacenter,Virtual Cluster,Container Labels,Container Info,Pod Info,Pod Labels,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Container Name,Current Nodes,Power State,Created By Kind,Created By Name,Current Size,Create Time,Container Restarts,Namespace Labels,Namespace CPU Request,Namespace CPU Limit,Namespace Memory Request,Namespace Memory Limit\n')
	for n in systems:
		for i in systems[n]:
			if i != 'namespace_labels' and i != 'cpu_limit' and i != 'cpu_request' and i != 'mem_limit' and i != 'mem_request':
				for j in systems[n][i]:
					# Ignores pod specific attributes as we only want to update items that are containers. 
					if i !='' and j != 'pod_info' and j != 'pod_labels' and j != 'owner_kind' and j != 'owner_name' and j != 'pod_name' and j != 'current_size' and j != 'creation_time':
						#determining the state and converting to string instead of number. 
						if systems[n][i][j]['state'] == 1:
							cstate = 'Terminated'
						else:
							cstate = 'Running'
						x = i
						if i == '<none>':
							x = systems[n][i][j]['pod_name']
						f.write(str(args.prom_addr) + ',' + n + ',' + x.replace(';','.') + ',' + j.replace(':','.') + ',Containers,' + str(args.prom_addr) + ',' + n + ',' + x + ',' + systems[n][i][j]['attr'] + ',' + systems[n][i][j]['con_info'] + ',' + systems[n][i]['pod_info'] + ',' + systems[n][i]['pod_labels'] + ',' + systems[n][i][j]['cpu_limit'] + ',' + systems[n][i][j]['cpu_request'] + ',' + systems[n][i][j]['mem_limit'] + ',' + systems[n][i][j]['mem_request'] + ',' + j + ',' + systems[n][i][j]['con_instance'][:-1] + ',' + cstate + ',' + systems[n][i]['owner_kind'] + ',' + systems[n][i]['owner_name'] + ',' + systems[n][i]['current_size'] + ',' + datetime.datetime.fromtimestamp(float(systems[n][i]['creation_time'])).strftime('%Y-%m-%d %H:%M:%S.%f')[:-3] + ',' + systems[n][i][j]['restarts'] + ',' + systems[n]['namespace_labels'] + ',' + systems[n]['cpu_request'] + ',' + systems[n]['cpu_limit'] + ',' + systems[n]['mem_request'] + ',' + systems[n]['mem_limit'] + '\n')
	f.close()

# Used to build metric string for metrics we get from kube state metrics as they have a similar query builds out the part that is repeated for them. 	
def getkubestatemetrics(systems,query,metric,current_time):
	query2 = str(args.aggregator) + '(' + query + ' * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)'
	data2 = multiDayCollect(query2,'result',current_time)

	for i in data2:
		if i['metric']['namespace'] in systems:
			if i['metric']['owner_name'] in systems[i['metric']['namespace']]:
				if i['metric']['container'] in systems[i['metric']['namespace']][i['metric']['owner_name']]:
					if args.mode == 'current':
						systems[i['metric']['namespace']][i['metric']['owner_name']][i['metric']['container']][metric] = i['value'][1]
					else:
						systems[i['metric']['namespace']][i['metric']['owner_name']][i['metric']['container']][metric] = i['values'][len(i['values'])-1][1]

	query2 = str(args.aggregator) + '(' + query + ' * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)'
	data2 = multiDayCollect(query2,'result',current_time)

	for i in data2:
		if i['metric']['namespace'] in systems:
			if i['metric']['pod'] in systems[i['metric']['namespace']]:
				if i['metric']['container'] in systems[i['metric']['namespace']][i['metric']['pod']]:
					if args.mode == 'current':
						systems[i['metric']['namespace']][i['metric']['pod']][i['metric']['container']][metric] = i['value'][1]
					else:
						systems[i['metric']['namespace']][i['metric']['pod']][i['metric']['container']][metric] = i['values'][len(i['values'])-1][1]
						
def getkubestatemetricspod(systems,query,namespace,service,metric,current_time):
	data2 = multiDayCollect(query,'result',current_time)
	for i in data2:
		if i['metric'][namespace] in systems:
			if i['metric'][service] in systems[i['metric'][namespace]]:
				if args.mode == 'current':
					systems[i['metric'][namespace]][i['metric'][service]][metric] = i['value'][1]
				else:
					systems[i['metric'][namespace]][i['metric'][service]][metric] = i['values'][len(i['values'])-1][1]
						
# For certain metrics we take the full results and build out multi-value tags to load all the details into different attributes
def getattributes(systems,data2,name1,name2,namespace,attribute):
	tempsystems = {}
	# loops through the data and makes sure haves the required names we require for controller and container as well as the pod being in the list we are collecting data for. 
	for i in data2:
		if name1 in i['metric']:
			if namespace in i['metric']:
				if i['metric'][namespace] in systems:
					if i['metric'][name1] in systems[i['metric'][namespace]]:
						#If system isn't in the temp array adds it.
						if i['metric'][namespace] not in tempsystems:
							tempsystems[i['metric'][namespace]] = {}
						if i['metric'][name1] not in tempsystems[i['metric'][namespace]]:
							tempsystems[i['metric'][namespace]][i['metric'][name1]] = {}
						if name2 in i['metric']:
							if i['metric'][name2] in systems[i['metric'][namespace]][i['metric'][name1]]:
								if i['metric'][name2] not in tempsystems[i['metric'][namespace]][i['metric'][name1]]:
									tempsystems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]] = {}
								for j in i['metric']:
									#if the attribute isn't in the temp list it adds it
									if j not in tempsystems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]:
										tempsystems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]][j] = i['metric'][j].replace(',',';')
									# if the attriubte is in the temp array then it appends the new value to it. 
									else:
										if i['metric'][j].replace(',',';') not in tempsystems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]][j]:
											tempsystems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]][j] += ';' + i['metric'][j].replace(',',';')
	
	# Loops through the temp array and combines all the different attributes found to form 1 giant multivalue attribute. 
	for n in tempsystems:
		for i in tempsystems[n]:
			for j in tempsystems[n][i]:
				attr = ''
				for k in tempsystems[n][i][j]:
					# if the key is over 250 then don't add as we can only add 256 and won't make sense. 
					if len(k) < 250:
						temp = tempsystems[n][i][j][k]
						#if the total for the key and valoue is under 256 then add it else we need to trim it so that it will fit as there is a 256 character limit on attributes. 
						if len(temp)+3+len(k) < 256:
							attr += k + ' : ' + temp + '|'
						else:
							templength = 256 - 3 - len(k)
							attr += k + ' : ' + temp[:templength] + '|'
					#If the attribute name is instance and we are loading to attribute "attr" then update con_instance with this value as well. 
					if k == 'instance':
						if attribute == 'attr':
							systems[n][i][j]['con_instance'] += tempsystems[n][i][j][k].replace(';','|') + '|'
					# If the name is pod then it sets the pod_name value as well for this system. 
					elif k == 'pod':
						systems[n][i][j]['pod_name'] = tempsystems[n][i][j][k]
				attr = attr[:-1]
				#updates the system array with the multivalue attribute based on name given.
				systems[n][i][j][attribute] = attr
							
# Similar function to the one above for containers just this builds the multi-value stirngs for the pod level attributes. 
def getattributespod(systems,data2,name1,namespace,attribute):
	tempsystems = {}
	for i in data2:
		if namespace in i['metric']:
			if name1 in i['metric']:
				if i['metric'][namespace] in systems:
					if i['metric'][name1] in systems[i['metric'][namespace]]:
						if i['metric'][namespace] not in tempsystems:
							tempsystems[i['metric'][namespace]] = {}
						if i['metric'][name1] not in tempsystems[i['metric'][namespace]]:
							tempsystems[i['metric'][namespace]][i['metric'][name1]] = {}
						for j in i['metric']:
							if j not in tempsystems[i['metric'][namespace]][i['metric'][name1]]:
								tempsystems[i['metric'][namespace]][i['metric'][name1]][j] = i['metric'][j].replace(',',';')
							else:
								if i['metric'][j].replace(',',';') not in tempsystems[i['metric'][namespace]][i['metric'][name1]][j]:
									tempsystems[i['metric'][namespace]][i['metric'][name1]][j] += ';' + i['metric'][j].replace(',',';')
				
	for n in tempsystems:
		for i in tempsystems[n]:
			attr = ''
			for j in tempsystems[n][i]:
				if len(j) < 250:
					temp = tempsystems[n][i][j]
					if len(temp)+3+len(j) < 256:
						attr += j + ' : ' + temp + '|'
					else:
						templength = 256 - 3 - len(j)
						attr += j + ' : ' + temp[:templength] + '|'
			attr = attr[:-1]
			systems[n][i][attribute] = attr
							
# Similar function to the one above for containers just this builds the multi-value stirngs for the pod level attributes. 
def getattributesnamespace(systems,data2,namespace,attribute):
	tempsystems = {}
	for i in data2:
		if namespace in i['metric']:
			if i['metric'][namespace] in systems:
				if i['metric'][namespace] not in tempsystems:
					tempsystems[i['metric'][namespace]] = {}
				for j in i['metric']:
					if j not in tempsystems[i['metric'][namespace]]:
						tempsystems[i['metric'][namespace]][j] = i['metric'][j].replace(',',';')
					else:
						if i['metric'][j].replace(',',';') not in tempsystems[i['metric'][namespace]][j]:
							tempsystems[i['metric'][namespace]][j] += ';' + i['metric'][j].replace(',',';')
				
	for n in tempsystems:
		attr = ''
		for j in tempsystems[n]:
			if len(j) < 250:
				temp = tempsystems[n][j]
				if len(temp)+3+len(j) < 256:
					attr += j + ' : ' + temp + '|'
				else:
					templength = 256 - 3 - len(j)
					attr += j + ' : ' + temp[:templength] + '|'
		attr = attr[:-1]
		systems[n][attribute] = attr
		
def main():
	# gets the current time so that all the different queries we run will be based off the same timestamp vs each one having a slightly different time as the script is run and keeps querying for current time. 
	current_time = datetime.datetime.utcnow()
	
	# If there is a config fil used check the parameters and update where needed. Note command line parameters that different then default wins. 
	if str(args.config_file) != 'NA':
		f=open(str(args.config_file), 'r')
		for line in f:
			info = line.split()
			if len(info) !=0:
				if info[0] == 'history' and args.history == '1':
					args.history = info[1]
				elif info[0] == 'prometheus_address' and str(args.prom_addr) == '':
					args.prom_addr = info[1]
				elif info[0] == 'prometheus_port' and str(args.prom_port) == '':
					args.prom_port = info[1]
				elif info[0] == 'timeout' and args.timeout == '3600':
					args.timeout = info[1]
				elif info[0] == 'collection' and args.collection == 'kubernetes':
					args.collection = info[1]
				elif info[0] == 'mode' and args.mode == 'current':
					args.mode = info[1]
				elif info[0] == 'prometheus_protocol' and args.protocol == 'http':
					args.protocol = info[1]
				elif info[0] == 'ssl_certificate_verify' and args.verify == 'True':
					args.verify = info[1]
				elif info[0] == 'aggregator' and args.aggregator == 'max':
					args.aggregator = info[1]
				elif info[0] == 'interval' and args.interval == 'days':
					args.interval = info[1]
				elif info[0] == 'interval_size' and args.interval_size == '1':
					args.interval_size = info[1]
				elif info[0] == 'debug' and args.debug == 'false':
					args.debug = info[1]
		f.close()
	
	# Write out version number to script this way we know each time what version of the file was run. 
	debug_log.write('Version 0.2.0\n')
	
	# This dictionary holds different parameters that adjust how the query works based on if use kubernetes or swarm. Note Kubernetes is only one should use as swarm may not be current.
	dc_settings = {}
	dc_settings['kubernetes'] = {}
	#These filters and group by are for the general cadvisor metrics. For the kube state metrics they are coded in line as it varies based on container and pod in each command. To see those ones can search for kube_pod and look at what each does as all those metrics start with kube_pod
	dc_settings['kubernetes']['grpby'] = 'instance,pod_name,namespace,container_name,owner_name,owner_kind'
	dc_settings['kubernetes']['filter'] = '{name!~"k8s_POD_.*"}'
	dc_settings['kubernetes']['ksm1'] = '('
	dc_settings['kubernetes']['ksm2'] = '* on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (owner_name,owner_kind,namespace,container_name)'
	dc_settings['kubernetes']['ksm2pod'] = '* on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (pod_name,namespace,container_name)'
	#for kubernetes this will build the name of the container as pod name__container name
	dc_settings['kubernetes']['name1'] = 'owner_name'
	dc_settings['kubernetes']['name2'] = 'container_name'
	dc_settings['kubernetes']['pod_name'] = 'pod_name'

	
	name1 = dc_settings[args.collection]['name1']
	name2 = dc_settings[args.collection]['name2']
	pod_name = dc_settings[args.collection]['pod_name']
	namespace = 'namespace'
	
	#containers
	# creates the dictionary that will hold all the containers we plan to collect data from for this run we do this so as we make different calls we have a consistent set of contaienrs from start to end vs collecting data for ones that may be missing config if just created. 
	systems={}
	#Query to get the amount of memory container has also used to build default info about the containers into Systems.
	query = str(args.aggregator) + dc_settings[args.collection]['ksm1'] + 'sum(container_spec_memory_limit_bytes' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ')/1024/1024 ' + dc_settings[args.collection]['ksm2'] + ''
	data2 = multiDayCollect(query,'result',current_time)
	# Logs the query if debug enabled.
	if str(args.debug) == 'true':
		debug_log.write(query)
		debug_log.write('\n')
	
	# Loops through the results and builds out the list of systems for ones that have the name1 and name2
	for i in data2:
		if namespace in i['metric']:
			if name1 in i['metric']:
				if i['metric'][namespace] not in systems:
					systems[i['metric'][namespace]]={}
					systems[i['metric'][namespace]]['namespace_labels'] = ''
					systems[i['metric'][namespace]]['cpu_limit'] = ''
					systems[i['metric'][namespace]]['cpu_request'] = ''
					systems[i['metric'][namespace]]['mem_limit'] = ''
					systems[i['metric'][namespace]]['mem_request'] = ''
				if i['metric'][name1] not in systems[i['metric'][namespace]]:
					# dumps out the results from the query to prometheus. 
					if str(args.debug) == 'true':
						debug_log.write('Initialize systems\n')
						debug_log.write(json.dumps(i['metric']))
						debug_log.write('\n')
					systems[i['metric'][namespace]][i['metric'][name1]]={}
					systems[i['metric'][namespace]][i['metric'][name1]]['pod_info'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]]['pod_labels'] = ''
					if args.collection == 'kubernetes':
						systems[i['metric'][namespace]][i['metric'][name1]]['owner_kind'] = i['metric']['owner_kind']
						systems[i['metric'][namespace]][i['metric'][name1]]['owner_name'] = i['metric']['owner_name']
					else:
						systems[i['metric'][namespace]][i['metric'][name1]]['owner_kind'] = ''
						systems[i['metric'][namespace]][i['metric'][name1]]['owner_name'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]]['current_size'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]]['creation_time'] = ''
				if name2 in i['metric']:
					systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]] = {}
					if args.mode == 'current':
						systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['memory'] = i['value'][1]
					else:
						systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['memory'] = i['values'][len(i['values'])-1][1]
					systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['attr'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['con_instance'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['con_info'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['cpu_limit'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['cpu_request'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['mem_limit'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['mem_request'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['restarts'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['pod_name'] = ''
					systems[i['metric'][namespace]][i['metric'][name1]][i['metric'][name2]]['state'] = 1
	
	#Query to get the amount of memory container has also used to build default info about the containers into Systems.
	query = str(args.aggregator) + dc_settings[args.collection]['ksm1'] + 'sum(container_spec_memory_limit_bytes' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ')/1024/1024 ' + dc_settings[args.collection]['ksm2pod'] + ''
	data2 = multiDayCollect(query,'result',current_time)
	# Logs the query if debug enabled.
	if str(args.debug) == 'true':
		debug_log.write(query)
		debug_log.write('\n')
	
	# Loops through the results and builds out the list of systems for ones that have the pod_name and name2
	for i in data2:
		if namespace in i['metric']:
			if pod_name in i['metric']:
				if i['metric'][namespace] not in systems:
					systems[i['metric'][namespace]]={}
					systems[i['metric'][namespace]]['namespace_labels'] = ''
					systems[i['metric'][namespace]]['cpu_limit'] = ''
					systems[i['metric'][namespace]]['cpu_request'] = ''
					systems[i['metric'][namespace]]['mem_limit'] = ''
					systems[i['metric'][namespace]]['mem_request'] = ''
				if i['metric'][pod_name] not in systems[i['metric'][namespace]]:
					# dumps out the results from the query to prometheus. 
					if str(args.debug) == 'true':
						debug_log.write('Initialize systems\n')
						debug_log.write(json.dumps(i['metric']))
						debug_log.write('\n')
					systems[i['metric'][namespace]][i['metric'][pod_name]]={}
					systems[i['metric'][namespace]][i['metric'][pod_name]]['pod_info'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]]['pod_labels'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]]['owner_kind'] = '<none>'
					systems[i['metric'][namespace]][i['metric'][pod_name]]['owner_name'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]]['current_size'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]]['creation_time'] = ''
				if name2 in i['metric']:
					systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]] = {}
					if args.mode == 'current':
						systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['memory'] = i['value'][1]
					else:
						systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['memory'] = i['values'][len(i['values'])-1][1]
					systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['attr'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['con_instance'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['con_info'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['cpu_limit'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['cpu_request'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['mem_limit'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['mem_request'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['restarts'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['pod_name'] = ''
					systems[i['metric'][namespace]][i['metric'][pod_name]][i['metric'][name2]]['state'] = 1
	
	
	# dumps what is in systems after the first call will be json data structure
	if str(args.debug) == 'true':			
		debug_log.write('Dump systems \n')
		debug_log.write(json.dumps(systems))
		debug_log.write('\n')
	
	#kube state metrics start
	if args.collection == 'kubernetes':
		#Collects CPU Limits
		query = 'sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000'
		getkubestatemetrics(systems,query,'cpu_limit',current_time)
		
		#Collects CPU Requests
		query = 'sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000'
		getkubestatemetrics(systems,query,'cpu_request',current_time)
		
		#Collects Memory Limit
		query = 'sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024'
		getkubestatemetrics(systems,query,'mem_limit',current_time)
		
		#Collects Memory Requests
		query = 'sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024'
		getkubestatemetrics(systems,query,'mem_request',current_time)
			
		#Collects Memory Requests
		query = 'sum(kube_pod_container_status_restarts_total) by (pod,namespace,container)'
		getkubestatemetrics(systems,query,'restarts',current_time)
				
		# Query for current state of the container if terminated or not.
		# This should only be a query as we want to get the current containers list that are showing terminated or not.
		state = str(args.aggregator) + '(sum(kube_pod_container_status_terminated) by (pod,namespace,container) * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)'
		data2 = metricCollect(state,'result','query')
		
		for i in data2:
			if i['metric']['namespace'] in systems:
				if i['metric']['owner_name'] in systems[i['metric'][namespace]]:
					if i['metric']['container'] in systems[i['metric'][namespace]][i['metric']['owner_name']]:
						systems[i['metric'][namespace]][i['metric']['owner_name']][i['metric']['container']]['state'] = i['value'][1]
			
		# This should only be a query as we want to get the current containers list that are showing terminated or not.
		state = str(args.aggregator) + '(sum(kube_pod_container_status_terminated) by (pod,namespace,container) * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)'
		data2 = metricCollect(state,'result','query')
		
		for i in data2:
			if i['metric']['namespace'] in systems:
				if i['metric']['pod'] in systems[i['metric'][namespace]]:
					if i['metric']['container'] in systems[i['metric'][namespace]][i['metric']['pod']]:
						systems[i['metric'][namespace]][i['metric']['pod']][i['metric']['container']]['state'] = i['value'][1]
			
	# kube state metrics end
				
	# Queries for the CPU Shares and writes out container_labels multi-value attribute. 
	query = '(sum(container_spec_cpu_shares' + dc_settings[args.collection]['filter'] + ') by (pod_name,namespace,container_name)) * on (namespace,pod_name,container_name) group_right container_spec_cpu_shares * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")'
	data2 = multiDayCollect(query,'result',current_time)
	getattributes(systems,data2,name1,name2,namespace,'attr')
	
	query = '(sum(container_spec_cpu_shares' + dc_settings[args.collection]['filter'] + ') by (pod_name,namespace,container_name)) * on (namespace,pod_name,container_name) group_right container_spec_cpu_shares * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")'
	data2 = multiDayCollect(query,'result',current_time)
	getattributes(systems,data2,pod_name,name2,namespace,'attr')
	
	
	if str(args.debug) == 'true':
		debug_log.write('Dump systems Additional Attributes \n')
		debug_log.write(json.dumps(systems))
		debug_log.write('\n')	
	
	#kube state metrics start
	if args.collection == 'kubernetes':
		# Query for the Container Info multi-value attribute
		query = 'sum(kube_pod_container_info) by (pod,namespace,container) * on (namespace,pod,container) group_right kube_pod_container_info * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}'
		data2 = multiDayCollect(query,'result',current_time)
		getattributes(systems,data2,'owner_name','container',namespace,'con_info')
		
		query = 'sum(kube_pod_container_info) by (pod,namespace,container) * on (namespace,pod,container) group_right kube_pod_container_info * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}'
		data2 = multiDayCollect(query,'result',current_time)
		getattributes(systems,data2,'pod','container',namespace,'con_info')
		
		if str(args.debug) == 'true':
			debug_log.write('Dump systems kube state \n')
			debug_log.write(json.dumps(systems))
			debug_log.write('\n')	
	
		# Gets the Pod info multi-value attribute
		query = 'sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_owner{owner_kind!="<none>"}'
		data2 = multiDayCollect(query,'result',current_time)
		getattributespod(systems,data2,'owner_name',namespace,'pod_info')
	    
		query = 'sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_owner{owner_kind="<none>"}'
		data2 = multiDayCollect(query,'result',current_time)
		getattributespod(systems,data2,'pod',namespace,'pod_info')
	    
		# Gets the pod Labels multi-value attribute
		query = 'sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_labels * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}'
		data2 = multiDayCollect(query,'result',current_time)
		getattributespod(systems,data2,'owner_name',namespace,'pod_labels')
		
		query = 'sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_labels * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}'
		data2 = multiDayCollect(query,'result',current_time)
		getattributespod(systems,data2,'pod',namespace,'pod_labels')
		
		# Gets the namespace Labels multi-value attribute
		query = 'kube_namespace_labels'
		data2 = multiDayCollect(query,'result',current_time)
		getattributesnamespace(systems,data2,namespace,'namespace_labels')
		
		# get the limits and requests of the namespace
		query = 'kube_limitrange'
		data2 = multiDayCollect(query,'result',current_time)
		for i in data2:
			if i['metric'][namespace] in systems:
				if i['metric']['constraint'] == 'default':
					if i['metric']['resource'] == 'cpu':
						systems[i['metric'][namespace]]['cpu_limit'] = i['value'][1]
					elif i['metric']['resource'] == 'memory':
						systems[i['metric'][namespace]]['mem_limit'] = str(int(i['value'][1])/1024/1024)
				elif i['metric']['constraint'] == 'defaultRequest':
					if i['metric']['resource'] == 'cpu':
						systems[i['metric'][namespace]]['cpu_request'] = i['value'][1]
					elif i['metric']['resource'] == 'memory':
						systems[i['metric'][namespace]]['mem_request'] = str(int(i['value'][1])/1024/1024)
			
		# Gets the number of replicas in the replica set. 
		query = 'kube_replicaset_spec_replicas'
		getkubestatemetricspod(systems,query,namespace,'replicaset','current_size',current_time)
		
		# Gets the number of replicas in the replication controller. 
		query = 'kube_replicationcontroller_spec_replicas'
		getkubestatemetricspod(systems,query,namespace,'replicationcontroller','current_size',current_time)
		
		# Gets the number of replicas in the daemon set. 
		query = 'kube_daemonset_status_number_available'
		getkubestatemetricspod(systems,query,namespace,'daemonset','current_size',current_time)
		
		# Gets the creation date. 
		query = str(args.aggregator) + '(kube_pod_created * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)'
		getkubestatemetricspod(systems,query,namespace,'owner_name','creation_time',current_time)
		
		query = str(args.aggregator) + '(kube_pod_created * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)'
		getkubestatemetricspod(systems,query,namespace,'pod','creation_time',current_time)
		
		
	# kube state metrics end
	
	# writes out the config and attributes files based on data from all the queries ran above. 
	writeConfig(systems,'CONTAINERS')
	writeAttributes(systems,'container')
		
	#workload metrics	
	count = 0
	#Loops through and collects the workload data starting with the newest data and working back to the oldest data this way if we have an issue we still get some data vs none as Prometheus if has issues is usually with collecting data from the older data.
	while count < int(args.history):
		count2 = count + int(args.interval_size)
		if args.interval == 'days':
			start = (current_time - datetime.timedelta(days=count2)).strftime("%Y-%m-%dT00:00:00.000Z")
			end = (current_time - datetime.timedelta(days=count)).strftime("%Y-%m-%dT23:00:00.000Z")
		elif args.interval == 'hours':
			start = (current_time - datetime.timedelta(hours=count2)).strftime("%Y-%m-%dT%H:00:00.000Z")
			end = (current_time - datetime.timedelta(hours=count)).strftime("%Y-%m-%dT%H:00:00.000Z")
		elif args.interval == 'minutes':
			start = (current_time - datetime.timedelta(minutes=count2)).strftime("%Y-%m-%dT%H:%M:00.000Z")
			end = (current_time - datetime.timedelta(minutes=count)).strftime("%Y-%m-%dT%H:%M:00.000Z")
		
		#Collect CPU millicores utilization
		cpu_metrics = str(args.aggregator) + dc_settings[args.collection]['ksm1'] + 'round(sum(rate(container_cpu_usage_seconds_total' + dc_settings[args.collection]['filter'] + '[5m])) by (' + dc_settings[args.collection]['grpby'] + ')*1000,1) ' + dc_settings[args.collection]['ksm2'] + '&start=' + start + '&end=' + end + '&step=5m'
		data2 = metricCollect(cpu_metrics,'result','query_range')
		cpu_metrics = str(args.aggregator) + dc_settings[args.collection]['ksm1'] + 'round(sum(rate(container_cpu_usage_seconds_total' + dc_settings[args.collection]['filter'] + '[5m])) by (' + dc_settings[args.collection]['grpby'] + ')*1000,1) ' + dc_settings[args.collection]['ksm2pod'] + '&start=' + start + '&end=' + end + '&step=5m'
		data2 += metricCollect(cpu_metrics,'result','query_range')
		writeWorkload(data2,systems,'cpu_mCores_workload' + str(start[:-5]).replace(":","."),'CPU Utilization in mCores',name1,name2,namespace)

		# Collect Memory utilization 
		mem_metrics = str(args.aggregator) + dc_settings[args.collection]['ksm1'] + 'sum(container_memory_usage_bytes' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ') ' + dc_settings[args.collection]['ksm2'] + '&start=' + start + '&end=' + end + '&step=5m'
		data2 = metricCollect(mem_metrics,'result','query_range')
		mem_metrics = str(args.aggregator) + dc_settings[args.collection]['ksm1'] + 'sum(container_memory_usage_bytes' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ') ' + dc_settings[args.collection]['ksm2pod'] + '&start=' + start + '&end=' + end + '&step=5m'
		data2 += metricCollect(mem_metrics,'result','query_range')
		writeWorkload(data2,systems,'mem_workload' + str(start[:-5]).replace(":","."),'Raw Mem Utilization',name1,name2,namespace)
						
		# Collect RSS that we load into active memory
		rss_metrics = str(args.aggregator) + dc_settings[args.collection]['ksm1'] + 'sum(container_memory_rss' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ') ' + dc_settings[args.collection]['ksm2'] + '&start=' + start + '&end=' + end + '&step=5m'
		data2 = metricCollect(rss_metrics,'result','query_range')
		rss_metrics = str(args.aggregator) + dc_settings[args.collection]['ksm1'] + 'sum(container_memory_rss' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ') ' + dc_settings[args.collection]['ksm2pod'] + '&start=' + start + '&end=' + end + '&step=5m'
		data2 += metricCollect(rss_metrics,'result','query_range')
		writeWorkload(data2,systems,'rss_workload' + str(start[:-5]).replace(":","."),'Actual Memory Utilization',name1,name2,namespace)
		
		#Collect disk IO bytes 
		disk_bytes_metrics = str(args.aggregator) + dc_settings[args.collection]['ksm1'] + 'sum(container_fs_usage_bytes' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ') ' + dc_settings[args.collection]['ksm2'] + '&start=' + start + '&end=' + end + '&step=5m'
		data2 = metricCollect(disk_bytes_metrics,'result','query_range')
		disk_bytes_metrics = str(args.aggregator) + dc_settings[args.collection]['ksm1'] + 'sum(container_fs_usage_bytes' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ') ' + dc_settings[args.collection]['ksm2pod'] + '&start=' + start + '&end=' + end + '&step=5m'
		data2 += metricCollect(disk_bytes_metrics,'result','query_range')
		writeWorkload(data2,systems,'disk_workload' + str(start[:-5]).replace(":","."),'Raw Disk Utilization',name1,name2,namespace)
		
		count += int(args.interval_size)
	
if __name__=="__main__":
    main()