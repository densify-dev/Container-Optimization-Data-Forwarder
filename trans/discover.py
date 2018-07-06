import requests
import datetime
import time
import argparse
import string
import subprocess
import sys

# Parsing inputted arguments from batch
parser = argparse.ArgumentParser()

parser.add_argument('-a', dest='prom_addr', default='',
                    help='Name of Prometheus server')
parser.add_argument('-p', dest='prom_port', default='',
                    help='Port of Prometheus server')
parser.add_argument('-d', dest='days', default='0',
                    help='Amount of days to receive data')
parser.add_argument('-f', dest='config_file', default='NA',
                    help='Config file to load settings from')
parser.add_argument('-t', dest='timeout', default='3600',
                    help='Timeout for querying Prometheus')
parser.add_argument('-c', dest='collection', default='kubernetes',
                    help='Data collection: swarm or kubernetes')
parser.add_argument('-m', dest='mode', default='current',
                    help='What containers to collect ones running now or all the ones that ran in last days: current or all')
args = parser.parse_args()

def metricCollect(metric,dataTag):
	resp = requests.get(url=metric, timeout=int(args.timeout))
	#print(metric)
	if resp.status_code != 200:
		print(metric)
		print(resp.status_code)
		print(resp.text)
		sys.exit(1)
	data = resp.json()
	data2 = data['data'][dataTag]
	return data2

def writeWorkload(data2,systems,file,property,name1,name2):
	f=open('./data/' + file + '.csv', 'w+')
	f.write('host_name,Datetime,' + property + '\n')
	for i in data2:
		if name2 in i['metric']:
			if name1 in i['metric']:
				if i['metric'][name2] !='':
					if i['metric'][name1] in systems:
						if i['metric'][name2] in systems[i['metric'][name1]]:
							for j in i['values']:
								f.write(i['metric'][name1] + '__' + i['metric'][name2].replace(':','..') + ',' + datetime.datetime.fromtimestamp(j[0]).strftime('%Y-%m-%d %H:%M:%S.%f')[:-3] + ',' + j[1] + '\n')
	f.close()

def writeWorkloadNetwork(data2,systems,file,property,instance,name1,name2):
	f=open('./data/' + file + '.csv', 'w+')
	f.write('HOSTNAME,PROPERTY,INSTANCE,DT,VAL\n')
	values = {}
	for i in data2:
		if name2 in i['metric']:
			if name1 in i['metric']:
				if i['metric'][name2] !='':
					if i['metric'][name1] in systems:
						if i['metric'][name2] in systems[i['metric'][name1]]:
							values[i['metric'][name2]]={}
							for j in i['values']:
								values[i['metric'][name2]][j[0]]=[]
								values[i['metric'][name2]][j[0]].append(j[1])
								f.write(i['metric'][name1] + '__' + i['metric'][name2].replace(':','..') + ',' + property + ',' + instance + ',' + datetime.datetime.fromtimestamp(j[0]).strftime('%Y-%m-%d %H:%M:%S.%f')[:-3] + ',' + j[1] + '\n')
	f.close()
	return values

def writeConfig(systems,benchmark,cpu_speed,type):
	f=open('./data/config.csv', 'w+')
	if type == 'CONTAINERS':
		f.write('host_name,HW Total Memory,OS Name,HW Manufacturer,HW Model,HW Serial Number\n')
	for i in systems:
		for j in systems[i]:
			if j !='' and j != 'pod_info' and j != 'pod_labels' and j != 'created_by_kind' and j != 'created_by_name':
				f.write(i + '__' + j.replace(':','..') + ',' + str(systems[i][j]['memory']) + ',Linux,' + type + ',' + systems[i][j]['namespace'] + ',' + systems[i][j]['namespace'] + '\n')
	f.close()
		
def writeAttributes(systems,prometheus_addr,type):
	f=open('./data/attributes.csv', 'w+')
	if type == 'container':
		f.write('host_name,Virtual Technology,Virtual Domain,Virtual Datacenter,Virtual Cluster,Container Labels,Container Info,Pod Info,Pod Labels,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Container Name,Original Parent,Power State,Created By Kind, Created By Name\n')
	for i in systems:
		for j in systems[i]:
			if i !='' and j != 'pod_info' and j != 'pod_labels' and j != 'created_by_kind' and j != 'created_by_name':
				if systems[i][j]['state'] == 1:
					cstate = 'Terminated'
				else:
					cstate = 'Running'
				f.write(i + '__' + j.replace(':','..') + ',Containers,' + prometheus_addr + ',' + systems[i][j]['namespace'] + ',' + i + ',' + systems[i][j]['attr'] + ',' + systems[i][j]['con_info'] + ',' + systems[i]['pod_info'] + ',' + systems[i]['pod_labels'] + ',' + systems[i][j]['cpu_limit'] + ',' + systems[i][j]['cpu_request'] + ',' + systems[i][j]['mem_limit'] + ',' + systems[i][j]['mem_request'] + ',' + j + ',' + systems[i][j]['con_instance'] + ',' + cstate + ',' + systems[i]['created_by_kind'] + ',' + systems[i]['created_by_name'] + '\n')
	f.close()
		
def getkubestatemetrics(systems,query,metric,prometheus_addr):
	data2 = []
	if args.mode == 'current':
		query2 = 'http://' + prometheus_addr + '/api/v1/query?query=' + query
		data2 += metricCollect(query2,'result')
	else:
		count = int(args.days)
		while count > -1:
			current_day = (datetime.datetime.today() - datetime.timedelta(days=count)).strftime("%Y-%m-%d")
			query2 = 'http://' + prometheus_addr + '/api/v1/query_range?query=' + query + '&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
			data2 += metricCollect(query2,'result')
			count -= 1
	
	for i in data2:
		if i['metric']['pod'] in systems:
			if i['metric']['container'] in systems[i['metric']['pod']]:
				if args.mode == 'current':
					systems[i['metric']['pod']][i['metric']['container']][metric] = i['value'][1]
				else:
					systems[i['metric']['pod']][i['metric']['container']][metric] = i['values'][len(i['values'])-1][1]
		
def main():
	if str(args.config_file) != 'NA':
		f=open(str(args.config_file), 'r')
		for line in f:
			info = line.split()
			if len(info) !=0:
				if info[0] == 'days' and args.days == '0':
					args.days = info[1]
				elif info[0] == 'promtheus_address' and str(args.prom_addr) == '':
					args.prom_addr = info[1]
				elif info[0] == 'promtheus_port' and str(args.prom_port) == '':
					args.prom_port = info[1]
				elif info[0] == 'timeout' and args.timeout == '3600':
					args.timeout = info[1]
				elif info[0] == 'collection' and args.collection == 'kubernetes':
					args.collection = info[1]
				elif info[0] == 'mode' and args.mode == 'current':
					args.mode = info[1]
		f.close()
	prometheus_addr = str(args.prom_addr) + ':' + str(args.prom_port) 
	cpu_speed = 2400
	benchmark = 44.1 # 441 per 10 core chip.
	
	dc_settings = {}
	dc_settings['kubernetes'] = {}
	#These filters and group by are for the general cadvisor metrics. For the kube state metrics they are coded in line as it varies based on container and pod in each command. To see those ones can search for kube_pod and look at what each does as all those metrics start with kube_pod
	dc_settings['kubernetes']['grpby'] = 'instance,pod_name,namespace,container_name,created_by_name,created_by_kind'
	dc_settings['kubernetes']['filter'] = '{name!~"k8s_POD_.*"}'
	#for kubernetes this will build the name of the container as pod name__container name
	dc_settings['kubernetes']['name1'] = 'pod_name'
	dc_settings['kubernetes']['name2'] = 'container_name'
	dc_settings['swarm'] = {}
	dc_settings['swarm']['grpby'] = 'name,instance,id'
	dc_settings['swarm']['filter'] = ''
	#for swarm this will build the name of the container as container name__instance name
	dc_settings['swarm']['name1'] = 'name'
	dc_settings['swarm']['name2'] = 'instance'
		
	#containers
	systems={}
	data2 =[]
	if args.mode == 'current':					
		mem_spec = 'http://' + prometheus_addr + '/api/v1/query?query=sum(container_spec_memory_limit_bytes' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ')/1024/1024'
		data2 += metricCollect(mem_spec,'result')
	else:
		count = int(args.days)
		while count > -1:
			current_day = (datetime.datetime.today() - datetime.timedelta(days=count)).strftime("%Y-%m-%d")
			mem_spec = 'http://' + prometheus_addr + '/api/v1/query_range?query=sum(container_spec_memory_limit_bytes' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ')/1024/1024&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
			data2 += metricCollect(mem_spec,'result')
			count -= 1
	
	for i in data2:
		if dc_settings[args.collection]['name1'] in i['metric']:
			if i['metric'][dc_settings[args.collection]['name1']] not in systems:
				systems[i['metric'][dc_settings[args.collection]['name1']]]={}
				systems[i['metric'][dc_settings[args.collection]['name1']]]['pod_info'] = ''
				systems[i['metric'][dc_settings[args.collection]['name1']]]['pod_labels'] = ''
				systems[i['metric'][dc_settings[args.collection]['name1']]]['created_by_kind'] = ''
				systems[i['metric'][dc_settings[args.collection]['name1']]]['created_by_name'] = ''
			if dc_settings[args.collection]['name2'] in i['metric']:
				systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]] = {}
				if args.collection == 'kubernetes':
					systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['namespace'] = i['metric']['namespace']
				else:
					systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['namespace'] = 'Default'
				if args.mode == 'current':
					systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['memory'] = i['value'][1]
				else:
					systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['memory'] = i['values'][len(i['values'])-1][1]
				systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['attr'] = ''
				systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['con_instance'] = ''
				systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['con_info'] = ''
				systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['cpu_limit'] = ''
				systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['cpu_request'] = ''
				systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['mem_limit'] = ''
				systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['mem_request'] = ''
				systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['state'] = 1

	#kube state metrics start
	if args.collection == 'kubernetes':
		query = 'sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000'
		getkubestatemetrics(systems,query,'cpu_limit',prometheus_addr)
		
		query = 'sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000'
		getkubestatemetrics(systems,query,'cpu_request',prometheus_addr)
		
		query = 'sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024'
		getkubestatemetrics(systems,query,'mem_limit',prometheus_addr)
		
		query = 'sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024'
		getkubestatemetrics(systems,query,'mem_request',prometheus_addr)
				
		# This should only be a query as we want to get the current containers list that are showing terminated or not.
		state = 'http://' + prometheus_addr + '/api/v1/query?query=sum(kube_pod_container_status_terminated) by (pod,namespace,container)'
		data2 = metricCollect(state,'result')
		
		for i in data2:
			if i['metric']['pod'] in systems:
				if i['metric']['container'] in systems[i['metric']['pod']]:
					systems[i['metric']['pod']][i['metric']['container']]['state'] = i['value'][1]
	else:
		# for swarm will just look at current value for memory to see what containers exist right now.
		# This should only be a query as we want to get the current containers list that are showing terminated or not.
		state = 'http://' + prometheus_addr + '/api/v1/query?query=sum(container_spec_memory_limit_bytes' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ')/1024/1024'
		data2 = metricCollect(state,'result')
		
		for i in data2:
			if dc_settings[args.collection]['name1'] in i['metric']:
				if dc_settings[args.collection]['name2'] in i['metric']:
					systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['state'] = 0
					
	# kube state metrics end
				
	writeConfig(systems,benchmark,cpu_speed,'CONTAINERS')
	
	# Additional Attributes?
	data2 = []
	if args.mode == 'current':					
		attr_spec = 'http://' + prometheus_addr + '/api/v1/query?query=(sum(container_spec_cpu_shares' + dc_settings[args.collection]['filter'] + ') by (pod_name,namespace,container_name)) * on (namespace,pod_name,container_name) group_right container_spec_cpu_shares'
		data2 += metricCollect(attr_spec,'result')
	else:
		count = int(args.days)
		while count > -1:
			current_day = (datetime.datetime.today() - datetime.timedelta(days=count)).strftime("%Y-%m-%d")
			attr_spec = 'http://' + prometheus_addr + '/api/v1/query_range?query=(sum(container_spec_cpu_shares' + dc_settings[args.collection]['filter'] + ') by (pod_name,namespace,container_name)) * on (namespace,pod_name,container_name) group_right container_spec_cpu_shares&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
			data2 += metricCollect(attr_spec,'result')
			count -= 1
				
	for i in data2:
		if dc_settings[args.collection]['name1'] in i['metric']:
			if dc_settings[args.collection]['name2'] in i['metric']:
				attr = ''
				for j in i['metric']:
					attr += j + ' : ' + i['metric'][j].replace(',',';') + '|'
					if j == 'instance':
						systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['con_instance'] = i['metric'][j]
				attr = attr[:-1]
				#print(attr)
				systems[i['metric'][dc_settings[args.collection]['name1']]][i['metric'][dc_settings[args.collection]['name2']]]['attr'] = attr
		
	#kube state metrics start
	if args.collection == 'kubernetes':
		data2 = []
		if args.mode == 'current':					
			attr_spec = 'http://' + prometheus_addr + '/api/v1/query?query=sum(kube_pod_container_info) by (pod,namespace,container) * on (namespace,pod,container) group_right kube_pod_container_info'
			data2 += metricCollect(attr_spec,'result')
		else:
			count = int(args.days)
			while count > -1:
				current_day = (datetime.datetime.today() - datetime.timedelta(days=count)).strftime("%Y-%m-%d")
				attr_spec = 'http://' + prometheus_addr + '/api/v1/query?query=sum(kube_pod_container_info) by (pod,namespace,container) * on (namespace,pod,container) group_right kube_pod_container_info&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
				data2 += metricCollect(attr_spec,'result')
				count -= 1
				
		for i in data2:
			if i['metric']['pod'] in systems:
				if i['metric']['container'] in systems[i['metric']['pod']]:
					attr = ''
					for j in i['metric']:
						attr += j + ' : ' + i['metric'][j].replace(',',';') + '|'
					attr = attr[:-1]
					systems[i['metric']['pod']][i['metric']['container']]['con_info'] = attr
		
		data2 = []
		if args.mode == 'current':					
			attr_spec = 'http://' + prometheus_addr + '/api/v1/query?query=sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_info'
			data2 += metricCollect(attr_spec,'result')
		else:
			count = int(args.days)
			while count > -1:
				current_day = (datetime.datetime.today() - datetime.timedelta(days=count)).strftime("%Y-%m-%d")
				attr_spec = 'http://' + prometheus_addr + '/api/v1/query?query=sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_info&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
				data2 += metricCollect(attr_spec,'result')
				count -= 1
				
		for i in data2:
			if i['metric']['pod'] in systems:
				attr = ''
				for j in i['metric']:
					if j == 'created_by_kind':
						systems[i['metric']['pod']]['created_by_kind'] = i['metric'][j]
					elif j == 'created_by_name':
						systems[i['metric']['pod']]['created_by_name'] = i['metric'][j]
					attr += j + ' : ' + i['metric'][j].replace(',',';') + '|'
				attr = attr[:-1]
				systems[i['metric']['pod']]['pod_info'] = attr
		
		data2 = []
		if args.mode == 'current':					
			attr_spec = 'http://' + prometheus_addr + '/api/v1/query?query=sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_labels'
			data2 += metricCollect(attr_spec,'result')
		else:
			count = int(args.days)
			while count > -1:
				current_day = (datetime.datetime.today() - datetime.timedelta(days=count)).strftime("%Y-%m-%d")
				attr_spec = 'http://' + prometheus_addr + '/api/v1/query?query=sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_labels&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
				data2 += metricCollect(attr_spec,'result')
				count -= 1
						
		for i in data2:
			if i['metric']['pod'] in systems:
				attr = ''
				for j in i['metric']:
					attr += j + ' : ' + i['metric'][j].replace(',',';') + '|'
				attr = attr[:-1]
				systems[i['metric']['pod']]['pod_labels'] = attr
	
	# kube state metrics end
	
	writeAttributes(systems,prometheus_addr,'container')
		
	#workload metrics	
	count = 0
	while count <= int(args.days):
		current_day = (datetime.datetime.today() - datetime.timedelta(days=count)).strftime("%Y-%m-%d")
	
		cpu_metrics = 'http://' + prometheus_addr + '/api/v1/query_range?query=round(sum(rate(container_cpu_usage_seconds_total' + dc_settings[args.collection]['filter'] + '[5m])) by (' + dc_settings[args.collection]['grpby'] + ')*1000,1)&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
		data2 = metricCollect(cpu_metrics,'result')
		writeWorkload(data2,systems,'cpu_mCores_workload' + current_day,'CPU Utilization in mCores',dc_settings[args.collection]['name1'],dc_settings[args.collection]['name2'])

		mem_metrics = 'http://' + prometheus_addr + '/api/v1/query_range?query=sum(container_memory_usage_bytes' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ')&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
		data2 = metricCollect(mem_metrics,'result')
		writeWorkload(data2,systems,'mem_workload' + current_day,'Raw Mem Utilization',dc_settings[args.collection]['name1'],dc_settings[args.collection]['name2'])
						
		rss_metrics = 'http://' + prometheus_addr + '/api/v1/query_range?query=sum(container_memory_rss' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ')&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
		data2 = metricCollect(rss_metrics,'result')
		writeWorkload(data2,systems,'rss_workload' + current_day,'Actual Memory Utilization',dc_settings[args.collection]['name1'],dc_settings[args.collection]['name2'])
		
		disk_bytes_metrics = 'http://' + prometheus_addr + '/api/v1/query_range?query=sum(container_fs_usage_bytes' + dc_settings[args.collection]['filter'] + ') by (' + dc_settings[args.collection]['grpby'] + ')&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
		data2 = metricCollect(disk_bytes_metrics,'result')
		writeWorkload(data2,systems,'disk_workload' + current_day,'Raw Disk Utilization',dc_settings[args.collection]['name1'],dc_settings[args.collection]['name2'])
							
		net_s_bytes_metrics = 'http://' + prometheus_addr + '/api/v1/query_range?query=sum(rate(container_network_transmit_bytes_total' + dc_settings[args.collection]['filter'] + '[5m])) by (' + dc_settings[args.collection]['grpby'] + ')&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
		data2 = metricCollect(net_s_bytes_metrics,'result')
		valuesSend = writeWorkloadNetwork(data2,systems,'net_bytes_s_workload' + current_day,'Network Interface Bytes Sent per sec','',dc_settings[args.collection]['name1'],dc_settings[args.collection]['name2'])
		
		net_r_bytes_metrics = 'http://' + prometheus_addr + '/api/v1/query_range?query=sum(rate(container_network_receive_bytes_total' + dc_settings[args.collection]['filter'] + '[5m])) by (' + dc_settings[args.collection]['grpby'] + ')&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
		data2 = metricCollect(net_r_bytes_metrics,'result')
		valuesReceived = writeWorkloadNetwork(data2,systems,'net_bytes_r_workload' + current_day,'Network Interface Bytes Received per sec','',dc_settings[args.collection]['name1'],dc_settings[args.collection]['name2'])
		
		f12=open('./data/net_bytes_workload' + current_day + '.csv', 'w+')
		f12.write('HOSTNAME,PROPERTY,INSTANCE,DT,VAL\n')
		for i in valuesSend:
			for j in valuesSend[i]:
				f12.write(i + ',Network Interface Bytes Total per sec,,' + datetime.datetime.fromtimestamp(j).strftime('%Y-%m-%d %H:%M:%S.%f')[:-3] + ',' + str(float(valuesReceived[i][j][0]) + float(valuesSend[i][j][0])) + '\n')
		f12.close()
		
		net_s_pkts_metrics = 'http://' + prometheus_addr + '/api/v1/query_range?query=sum(rate(container_network_transmit_packets_total' + dc_settings[args.collection]['filter'] + '[5m])) by (' + dc_settings[args.collection]['grpby'] + ')&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
		data2 = metricCollect(net_s_pkts_metrics,'result')
		valuesSend = writeWorkloadNetwork(data2,systems,'net_pkts_s_workload' + current_day,'Network Interface Packets Sent per sec','',dc_settings[args.collection]['name1'],dc_settings[args.collection]['name2'])
												
		net_r_pkts_metrics = 'http://' + prometheus_addr + '/api/v1/query_range?query=sum(rate(container_network_receive_packets_total' + dc_settings[args.collection]['filter'] + '[5m])) by (' + dc_settings[args.collection]['grpby'] + ')&start=' + current_day + 'T00:00:00.000Z&end=' + current_day + 'T23:59:59.000Z&step=5m'
		data2 = metricCollect(net_r_pkts_metrics,'result')
		valuesReceived = writeWorkloadNetwork(data2,systems,'net_pkts_r_workload' + current_day,'Network Interface Packets Received per sec','',dc_settings[args.collection]['name1'],dc_settings[args.collection]['name2'])
		
		f13=open('./data/net_pkts_workload' + current_day +'.csv', 'w+')
		f13.write('HOSTNAME,PROPERTY,INSTANCE,DT,VAL\n')
		for i in valuesSend:
			for j in valuesSend[i]:
				f13.write(i + ',Network Interface Packets per sec,,' + datetime.datetime.fromtimestamp(j).strftime('%Y-%m-%d %H:%M:%S.%f')[:-3] + ',' + str(float(valuesReceived[i][j][0]) + float(valuesSend[i][j][0])) + '\n')
		f13.close()
		
		count += 1
		
	#Replica Set data 
	
	
if __name__=="__main__":
    main()