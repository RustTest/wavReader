
import {check} from 'k6';
import wavreader from 'k6/x/wavreader';

let client = wavreader.createClient({url: `pulsar://${__ENV.PULSAR_ADDR}`})
let producer = wavreader.createProducer(client, {topic: __ENV.PULSAR_TOPIC})
const audioFileLocation ="/Users/prasadchandrasekaran/code/lasthope/wavReader/652-130726-combined.wav";

export default function() {
  // 3. VU code

  console.log(`starting the load for voxflo`);
  let err = wavreader.publish(producer, null, {}, false, audioFileLocation,333);
  check(err, {
  "is send": err => err == null
   })
}

export function setup() {


}

export function teardown() {
  // 4. teardown code
  wavreader.closeClient(client)
  wavreader.closeProducer(producer)
  console.log("teardown!!")
}
