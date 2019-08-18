import re
import sys
from datetime import datetime, timedelta
from elasticsearch_dsl import Search, Q
from tests.queries import ES, get_latest_layer

dt = datetime.now()
todaydate = dt.strftime("%Y.%m.%d")
current_index = 'kubernetes_cluster-' + todaydate

LAYERS_PER_EPOCH = 5


class ESQuery():

    def __init__(self, indx, namespace):
        self.index = indx
        self.namespace = namespace
        self.es = ES().get_search_api()

    def _query(self, fields, desc=False):
        fltr = Q("match_phrase", kubernetes__namespace_name=self.namespace)
        for f in fields:
            fltr = fltr & Q("match_phrase", **{f: fields[f]})
        if desc:
            s = Search(index=self.index, using=self.es).query('bool', filter=[fltr]).sort('-T')
        else:
            s = Search(index=self.index, using=self.es).query('bool', filter=[fltr]).sort('T')
        return s

    def get_latest(self, fields):
        se = self._query(fields, desc=True)
        response = se.execute()
        return response.hits[0]

    def get_first(self, fields):
        se = self._query(fields)
        response = se.execute()
        return response.hits[0]

    def get_all(self, fields):
        se = self._query(fields)
        return list(se.scan())

    def count(self, fields):
        se = self._query(fields)
        response = se.execute()
        return response.hits.total


def get_latest_layer_released_ticks(namespace):

    block_fields = {"M": "release tick"}
    hit = ESQuery(current_index, namespace).get_latest(block_fields)
    return hit.layer_id,  hit.T


def get_layer_released_ticks(namespace, layer_id=False):
    if not layer_id:
        block_fields = {"M": "release tick"}
    else:
        block_fields = {"M": "release tick", "layer_id": layer_id}
    hits = ESQuery(current_index, namespace).get_all(block_fields)
    return hits


def count_atx_published_in_epoch(namespace, epoch_id):
    fields = {"M": "atx published", "epoch_id": epoch_id}
    num_of_hits = ESQuery(current_index, namespace).count(fields)
    return num_of_hits


def get_atx_published_in_epoch(namespace, epoch_id):
    fields = {"M": "atx published", "epoch_id": epoch_id}
    hits = ESQuery(current_index, namespace).get_all(fields)
    return hits


def main():

    namespace = sys.argv[1]
    expected_atxs = int(sys.argv[2])

    # 1. Calculate epoch to monitor
    layer, _ = get_latest_layer_released_ticks(namespace)
    epoch_to_monitor = int(layer/LAYERS_PER_EPOCH)-1
    print("Epoch to monitor: {0}".format(epoch_to_monitor))

    # 2. Count "atx published"
    atxs = get_atx_published_in_epoch(namespace, epoch_to_monitor)
    num_published_atx = len(atxs)
    assert(num_published_atx == expected_atxs), "Number of ATX published: %d  not as expected %d" % (num_published_atx, expected_atxs)
    print("Num of published atx: {0}".format(num_published_atx))

    start_epoch_layer = (epoch_to_monitor*LAYERS_PER_EPOCH)+1
    end_epoch_layer = (epoch_to_monitor+1)*LAYERS_PER_EPOCH

    start_epoch_layer_release_tick_hits = get_layer_released_ticks(namespace, layer_id=start_epoch_layer)
    release_times = [datetime.strptime(x.T.replace("T", " ", ).replace("Z", ""), "%Y-%m-%d %H:%M:%S.%f") for x in start_epoch_layer_release_tick_hits]
    release_times.sort()
    start = release_times[0]

    end_epoch_layer_release_tick_hits = get_layer_released_ticks(namespace, layer_id=end_epoch_layer)
    release_times = [datetime.strptime(x.T.replace("T", " ", ).replace("Z", ""), "%Y-%m-%d %H:%M:%S.%f") for x in end_epoch_layer_release_tick_hits]
    release_times.sort()
    end = release_times[-1]

    print("Start: {0} End: {1}".format(start, end))

    for atx in atxs:
        atx_time = datetime.strptime(atx.T.replace("T", " ", ).replace("Z", ""), "%Y-%m-%d %H:%M:%S.%f")
        print("ATX time: {0}".format(atx_time))
        assert (atx_time > start and atx_time <= end), "ATX not in time delta"


if __name__ == "__main__":
    main()