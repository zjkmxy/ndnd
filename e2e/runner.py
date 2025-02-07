import random
import os

from types import FunctionType

from mininet.log import setLogLevel
from minindn.minindn import Minindn

import test_001

def run(scenario: FunctionType, **kwargs) -> None:
    try:
        scenario(ndn, **kwargs)

        # Call all cleanups without stopping the network
        # This ensures we don't recreate the network for each test
        for cleanup in reversed(ndn.cleanups):
            cleanup()
    except Exception as e:
        ndn.stop()
        raise e
    finally:
        # kill everything we started just in case ...
        os.system('pkill -9 ndnd')
        os.system('pkill -9 nfd')

if __name__ == '__main__':
    setLogLevel('info')
    random.seed(0)

    Minindn.cleanUp()
    Minindn.verifyDependencies()

    ndn = Minindn()
    ndn.start()

    run(test_001.scenario_ndnd_fw)
    run(test_001.scenario_nfd)

    ndn.stop()
