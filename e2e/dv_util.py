import time

from mininet.log import info
from mininet.node import Node

from minindn.minindn import Minindn
from minindn.apps.app_manager import AppManager

from dv import NDNd_DV, DEFAULT_NETWORK

def setup(ndn: Minindn, network=DEFAULT_NETWORK) -> None:
    time.sleep(1) # wait for fw to start

    NDNd_DV.init_trust()
    info('Starting ndn-dv on nodes\n')
    AppManager(ndn, ndn.net.hosts, NDNd_DV, network=network)

def converge(nodes: list[Node], deadline=60, network=DEFAULT_NETWORK) -> int:
    info('Waiting for routing to converge\n')
    start = time.time()
    while time.time() - start < deadline:
        time.sleep(1)

        converged = True
        for node in nodes:
            routes = node.cmd('ndnd fw route-list')
            for other in nodes:
                if f'{network}/{other.name}/32=DV' not in routes:
                    info(f'Routing not converged on {node.name}\n')
                    converged = False
                    break # break out of inner loop
            if not converged:
                break # break out of outer loop
        if converged:
            total = round(time.time() - start)
            info(f'Routing converged in {total} seconds\n')
            return total

    raise Exception('Routing did not converge')
