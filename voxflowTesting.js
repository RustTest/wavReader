
import {check} from 'k6';
import wavreader from 'k6/x/wavreader';

let client = wavreader.createClient({url: `pulsar://${__ENV.PULSAR_ADDR}`})
const audioFileLocation ="/Users/prasadchandrasekaran/code/lasthope/wavReader/652-130726-combined.wav";

export default function() {
  // 3. VU code
  let res = callControllerForTopic();
  console.log(`starting the load for voxflo`);
  let producer = wavreader.createProducer(client, {topic: res})
  let err = wavreader.publish(producer, null, {}, false, audioFileLocation,333);
  check(err, {
  "is send": err => err == null
   })
}

function callControllerForTopic()  {
  const payload = JSON.stringify({
    stream_infra: 'Pulsar'
  });
  const headers = { 'Content-Type': 'application/json' };
  const res = http.post('http://172.31.12.28:9001/flows', payload, { headers });
  check(res, {
    'Post status is 200': (r) => res.status === 200,
    'Post Content-Type header': (r) => res.headers['Content-Type'] === 'application/json',
    'Post response name': (r) => res.status === 200 && res.json().json.name === 'lorem',
  });
  return res;
}


export function setup() {


}

export function teardown() {
  // 4. teardown code
  wavreader.closeClient(client)
  wavreader.closeProducer(producer)
  console.log("teardown!!")
}
